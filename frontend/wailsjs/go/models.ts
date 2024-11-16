export namespace data {
	
	export class DataSource {
	    id: string;
	    peer_id: string;
	    reputation: number;
	    data_types: string[];
	    available_symbols: string[];
	    // Go type: time
	    data_range_start: any;
	    // Go type: time
	    data_range_end: any;
	    // Go type: time
	    last_update: any;
	    reliability: number;
	
	    static createFrom(source: any = {}) {
	        return new DataSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.peer_id = source["peer_id"];
	        this.reputation = source["reputation"];
	        this.data_types = source["data_types"];
	        this.available_symbols = source["available_symbols"];
	        this.data_range_start = this.convertValues(source["data_range_start"], null);
	        this.data_range_end = this.convertValues(source["data_range_end"], null);
	        this.last_update = this.convertValues(source["last_update"], null);
	        this.reliability = source["reliability"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DividendData {
	    id: string;
	    symbol: string;
	    // Go type: time
	    timestamp: any;
	    source: string;
	    data_type: string;
	    validation_score: number;
	    up_votes: number;
	    down_votes: number;
	    metadata: {[key: string]: string};
	    amount: number;
	    currency: string;
	    // Go type: time
	    ex_date: any;
	    // Go type: time
	    pay_date: any;
	    // Go type: time
	    record_date: any;
	    // Go type: time
	    payment_date: any;
	    // Go type: time
	    declaration_date: any;
	    frequency: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new DividendData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.symbol = source["symbol"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.source = source["source"];
	        this.data_type = source["data_type"];
	        this.validation_score = source["validation_score"];
	        this.up_votes = source["up_votes"];
	        this.down_votes = source["down_votes"];
	        this.metadata = source["metadata"];
	        this.amount = source["amount"];
	        this.currency = source["currency"];
	        this.ex_date = this.convertValues(source["ex_date"], null);
	        this.pay_date = this.convertValues(source["pay_date"], null);
	        this.record_date = this.convertValues(source["record_date"], null);
	        this.payment_date = this.convertValues(source["payment_date"], null);
	        this.declaration_date = this.convertValues(source["declaration_date"], null);
	        this.frequency = source["frequency"];
	        this.type = source["type"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EODData {
	    id: string;
	    symbol: string;
	    // Go type: time
	    timestamp: any;
	    source: string;
	    data_type: string;
	    validation_score: number;
	    up_votes: number;
	    down_votes: number;
	    metadata: {[key: string]: string};
	    open: number;
	    high: number;
	    low: number;
	    close: number;
	    volume: number;
	    adjusted_close: number;
	    // Go type: time
	    date: any;
	
	    static createFrom(source: any = {}) {
	        return new EODData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.symbol = source["symbol"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.source = source["source"];
	        this.data_type = source["data_type"];
	        this.validation_score = source["validation_score"];
	        this.up_votes = source["up_votes"];
	        this.down_votes = source["down_votes"];
	        this.metadata = source["metadata"];
	        this.open = source["open"];
	        this.high = source["high"];
	        this.low = source["low"];
	        this.close = source["close"];
	        this.volume = source["volume"];
	        this.adjusted_close = source["adjusted_close"];
	        this.date = this.convertValues(source["date"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MarketData {
	    id: string;
	    symbol: string;
	    price: number;
	    volume: number;
	    // Go type: time
	    timestamp: any;
	    source: string;
	    data_type: string;
	    signatures: {[key: string]: uint8[]};
	    metadata?: {[key: string]: string};
	    validation_score: number;
	    hash: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new MarketData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.symbol = source["symbol"];
	        this.price = source["price"];
	        this.volume = source["volume"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
	        this.source = source["source"];
	        this.data_type = source["data_type"];
	        this.signatures = source["signatures"];
	        this.metadata = source["metadata"];
	        this.validation_score = source["validation_score"];
	        this.hash = source["hash"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MarketDataFilter {
	
	
	    static createFrom(source: any = {}) {
	        return new MarketDataFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Peer {
	    id: string;
	    address: string;
	    public_key: number[];
	    reputation: number;
	    // Go type: time
	    last_seen: any;
	    is_authority: boolean;
	    roles: string[];
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    status: string;
	    metadata?: {[key: string]: any};
	
	    static createFrom(source: any = {}) {
	        return new Peer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.address = source["address"];
	        this.public_key = source["public_key"];
	        this.reputation = source["reputation"];
	        this.last_seen = this.convertValues(source["last_seen"], null);
	        this.is_authority = source["is_authority"];
	        this.roles = source["roles"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.status = source["status"];
	        this.metadata = source["metadata"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class ServerStatus {
	    running: boolean;
	    databaseConnected: boolean;
	    p2pHostRunning: boolean;
	    scriptMgrRunning: boolean;
	    embeddedDbRunning: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServerStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.databaseConnected = source["databaseConnected"];
	        this.p2pHostRunning = source["p2pHostRunning"];
	        this.scriptMgrRunning = source["scriptMgrRunning"];
	        this.embeddedDbRunning = source["embeddedDbRunning"];
	    }
	}

}

