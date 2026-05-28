package host

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"p2p_market_data/pkg/data"

	libp2pCrypto "github.com/libp2p/go-libp2p/core/crypto"
	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
)

const streamAuthWindow = 5 * time.Minute

type requestAuthTracker struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

func newRequestAuthTracker() *requestAuthTracker {
	return &requestAuthTracker{
		seen: make(map[string]time.Time),
	}
}

func (t *requestAuthTracker) accept(peerID, nonce string, now time.Time) error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	for key, expiresAt := range t.seen {
		if !expiresAt.After(now) {
			delete(t.seen, key)
		}
	}

	key := peerID + ":" + nonce
	if _, exists := t.seen[key]; exists {
		return fmt.Errorf("replayed request nonce")
	}
	t.seen[key] = now.Add(streamAuthWindow * 2)
	return nil
}

func (h *Host) signDataRequest(req *data.DataRequest) error {
	if h == nil || h.host == nil {
		return fmt.Errorf("host not initialized")
	}
	nonce, err := generateAuthNonce()
	if err != nil {
		return err
	}
	req.RequesterPeerID = h.host.ID().String()
	pubKey, err := libp2pCrypto.MarshalPublicKey(h.host.Peerstore().PubKey(h.host.ID()))
	if err != nil {
		return fmt.Errorf("marshaling requester public key: %w", err)
	}
	req.RequesterPubKey = pubKey
	req.RequestedAt = time.Now().UTC().Unix()
	req.Nonce = nonce
	req.Signature = nil

	payload, err := dataRequestSigningPayload(*req)
	if err != nil {
		return err
	}
	privKey := h.host.Peerstore().PrivKey(h.host.ID())
	if privKey == nil {
		return fmt.Errorf("private key not found for local peer")
	}
	signature, err := privKey.Sign(payload)
	if err != nil {
		return fmt.Errorf("signing data request: %w", err)
	}
	req.Signature = signature
	return nil
}

func (h *Host) verifyDataRequest(remotePeer libp2pPeer.ID, req *data.DataRequest) error {
	if h == nil || h.host == nil {
		return fmt.Errorf("host not initialized")
	}
	if req.RequesterPeerID != remotePeer.String() {
		return fmt.Errorf("requester peer mismatch")
	}
	if req.RequestedAt == 0 {
		return fmt.Errorf("request timestamp is required")
	}
	if !withinAuthWindow(time.Unix(req.RequestedAt, 0), time.Now().UTC()) {
		return fmt.Errorf("request timestamp is outside the allowed window")
	}
	if len(req.Nonce) < 16 {
		return fmt.Errorf("request nonce is required")
	}
	if len(req.Signature) == 0 {
		return fmt.Errorf("request signature is required")
	}

	pubKey, err := publicKeyForPeer(h, remotePeer, req.RequesterPubKey, "requester")
	if err != nil {
		return err
	}
	payload, err := dataRequestSigningPayload(*req)
	if err != nil {
		return err
	}
	ok, err := pubKey.Verify(payload, req.Signature)
	if err != nil {
		return fmt.Errorf("verifying request signature: %w", err)
	}
	if !ok {
		return fmt.Errorf("request signature verification failed")
	}
	tracker := h.ensureRequestAuthTracker()
	return tracker.accept(remotePeer.String(), req.Nonce, time.Now().UTC())
}

func (h *Host) ensureRequestAuthTracker() *requestAuthTracker {
	h.mu.RLock()
	tracker := h.requestAuth
	h.mu.RUnlock()
	if tracker != nil {
		return tracker
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.requestAuth == nil {
		h.requestAuth = newRequestAuthTracker()
	}
	return h.requestAuth
}

func (h *Host) signDataResponse(resp *dataResponse, req data.DataRequest) error {
	if h == nil || h.host == nil {
		return fmt.Errorf("host not initialized")
	}
	resp.ResponderPeerID = h.host.ID().String()
	pubKey, err := libp2pCrypto.MarshalPublicKey(h.host.Peerstore().PubKey(h.host.ID()))
	if err != nil {
		return fmt.Errorf("marshaling responder public key: %w", err)
	}
	resp.ResponderPubKey = pubKey
	resp.RespondedAt = time.Now().UTC().Unix()
	resp.Nonce = req.Nonce
	resp.Signature = nil
	if resp.Error == "" {
		checksum, err := responseChecksum(*resp)
		if err != nil {
			return err
		}
		resp.Checksum = checksum
	}

	payload, err := dataResponseSigningPayload(*resp)
	if err != nil {
		return err
	}
	privKey := h.host.Peerstore().PrivKey(h.host.ID())
	if privKey == nil {
		return fmt.Errorf("private key not found for local peer")
	}
	signature, err := privKey.Sign(payload)
	if err != nil {
		return fmt.Errorf("signing data response: %w", err)
	}
	resp.Signature = signature
	return nil
}

func (h *Host) verifyDataResponse(remotePeer libp2pPeer.ID, request data.DataRequest, resp *dataResponse) error {
	if h == nil || h.host == nil {
		return fmt.Errorf("host not initialized")
	}
	if resp.ResponderPeerID != remotePeer.String() {
		return fmt.Errorf("responder peer mismatch")
	}
	if resp.Nonce != request.Nonce {
		return fmt.Errorf("response nonce mismatch")
	}
	if resp.RespondedAt == 0 {
		return fmt.Errorf("response timestamp is required")
	}
	if !withinAuthWindow(time.Unix(resp.RespondedAt, 0), time.Now().UTC()) {
		return fmt.Errorf("response timestamp is outside the allowed window")
	}
	if len(resp.Signature) == 0 {
		return fmt.Errorf("response signature is required")
	}

	pubKey, err := publicKeyForPeer(h, remotePeer, resp.ResponderPubKey, "responder")
	if err != nil {
		return err
	}
	payload, err := dataResponseSigningPayload(*resp)
	if err != nil {
		return err
	}
	ok, err := pubKey.Verify(payload, resp.Signature)
	if err != nil {
		return fmt.Errorf("verifying response signature: %w", err)
	}
	if !ok {
		return fmt.Errorf("response signature verification failed")
	}
	return nil
}

func publicKeyForPeer(h *Host, remotePeer libp2pPeer.ID, supplied []byte, role string) (libp2pCrypto.PubKey, error) {
	if len(supplied) > 0 {
		pubKey, err := libp2pCrypto.UnmarshalPublicKey(supplied)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling %s public key: %w", role, err)
		}
		derivedPeer, err := libp2pPeer.IDFromPublicKey(pubKey)
		if err != nil {
			return nil, fmt.Errorf("deriving %s peer id from public key: %w", role, err)
		}
		if derivedPeer != remotePeer {
			return nil, fmt.Errorf("%s public key does not match peer id", role)
		}
		return pubKey, nil
	}

	pubKey := h.host.Peerstore().PubKey(remotePeer)
	if pubKey != nil {
		return pubKey, nil
	}
	extracted, err := remotePeer.ExtractPublicKey()
	if err != nil {
		return nil, fmt.Errorf("public key not found for %s: %w", role, err)
	}
	return extracted, nil
}

func dataRequestSigningPayload(req data.DataRequest) ([]byte, error) {
	req.Signature = nil
	content, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling data request signing payload: %w", err)
	}
	return content, nil
}

func dataResponseSigningPayload(resp dataResponse) ([]byte, error) {
	resp.Signature = nil
	content, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshaling data response signing payload: %w", err)
	}
	return content, nil
}

func generateAuthNonce() (string, error) {
	var content [16]byte
	if _, err := rand.Read(content[:]); err != nil {
		return "", fmt.Errorf("generating request nonce: %w", err)
	}
	return hex.EncodeToString(content[:]), nil
}

func withinAuthWindow(ts, now time.Time) bool {
	if ts.After(now.Add(streamAuthWindow)) {
		return false
	}
	return ts.After(now.Add(-streamAuthWindow))
}
