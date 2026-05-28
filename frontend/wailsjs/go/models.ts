export namespace data {
	
	export class DataRequest {
	    type: string;
	    symbol: string;
	    start_date: string;
	    end_date: string;
	    granularity: string;
	
	    static createFrom(source: any = {}) {
	        return new DataRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.symbol = source["symbol"];
	        this.start_date = source["start_date"];
	        this.end_date = source["end_date"];
	        this.granularity = source["granularity"];
	    }
	}
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
	    metadata: Record<string, string>;
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
	    metadata: Record<string, string>;
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
	export class InsiderTrade {
	    id: string;
	    symbol: string;
	    // Go type: time
	    timestamp: any;
	    source: string;
	    data_type: string;
	    validation_score: number;
	    up_votes: number;
	    down_votes: number;
	    metadata: Record<string, string>;
	    insider_name: string;
	    insider_title: string;
	    trade_type: string;
	    // Go type: time
	    trade_date: any;
	    position: string;
	    shares: number;
	    price_per_share: number;
	    value: number;
	    transaction_type: string;
	
	    static createFrom(source: any = {}) {
	        return new InsiderTrade(source);
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
	        this.insider_name = source["insider_name"];
	        this.insider_title = source["insider_title"];
	        this.trade_type = source["trade_type"];
	        this.trade_date = this.convertValues(source["trade_date"], null);
	        this.position = source["position"];
	        this.shares = source["shares"];
	        this.price_per_share = source["price_per_share"];
	        this.value = source["value"];
	        this.transaction_type = source["transaction_type"];
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
	    signatures: Record<string, Array<number>>;
	    metadata?: Record<string, string>;
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
	    Symbol: string;
	    MinPrice?: number;
	    MaxPrice?: number;
	    // Go type: time
	    FromTime?: any;
	    // Go type: time
	    ToTime?: any;
	    Source: string;
	    DataType: string;
	    Limit: number;
	    Offset: number;
	
	    static createFrom(source: any = {}) {
	        return new MarketDataFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Symbol = source["Symbol"];
	        this.MinPrice = source["MinPrice"];
	        this.MaxPrice = source["MaxPrice"];
	        this.FromTime = this.convertValues(source["FromTime"], null);
	        this.ToTime = this.convertValues(source["ToTime"], null);
	        this.Source = source["Source"];
	        this.DataType = source["DataType"];
	        this.Limit = source["Limit"];
	        this.Offset = source["Offset"];
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
	    metadata?: Record<string, any>;
	
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
	export class SplitData {
	    id: string;
	    symbol: string;
	    // Go type: time
	    timestamp: any;
	    source: string;
	    data_type: string;
	    validation_score: number;
	    up_votes: number;
	    down_votes: number;
	    metadata: Record<string, string>;
	    split_ratio: number;
	    // Go type: time
	    announcement_date: any;
	    // Go type: time
	    ex_date: any;
	    old_shares: number;
	    new_shares: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new SplitData(source);
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
	        this.split_ratio = source["split_ratio"];
	        this.announcement_date = this.convertValues(source["announcement_date"], null);
	        this.ex_date = this.convertValues(source["ex_date"], null);
	        this.old_shares = source["old_shares"];
	        this.new_shares = source["new_shares"];
	        this.status = source["status"];
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
	
	export class ActiveTransfer {
	    id: string;
	    type: string;
	    symbol: string;
	    source: string;
	    destination: string;
	    progress: number;
	    status: string;
	    startTime: string;
	    endTime?: string;
	    size: number;
	    speed: number;
	
	    static createFrom(source: any = {}) {
	        return new ActiveTransfer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.symbol = source["symbol"];
	        this.source = source["source"];
	        this.destination = source["destination"];
	        this.progress = source["progress"];
	        this.status = source["status"];
	        this.startTime = source["startTime"];
	        this.endTime = source["endTime"];
	        this.size = source["size"];
	        this.speed = source["speed"];
	    }
	}
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
	export class AppHealthDiagnostics {
	    generatedAt: string;
	    status: ServerStatus;
	    databaseUrl: string;
	    requiredTables: Record<string, boolean>;
	    p2pHostId: string;
	    p2pListenAddresses: string[];
	    connectedPeers: string[];
	    scriptManagerRunning: boolean;
	    pythonRuntime: string;
	    latestTransferErrors: string[];
	
	    static createFrom(source: any = {}) {
	        return new AppHealthDiagnostics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generatedAt = source["generatedAt"];
	        this.status = this.convertValues(source["status"], ServerStatus);
	        this.databaseUrl = source["databaseUrl"];
	        this.requiredTables = source["requiredTables"];
	        this.p2pHostId = source["p2pHostId"];
	        this.p2pListenAddresses = source["p2pListenAddresses"];
	        this.connectedPeers = source["connectedPeers"];
	        this.scriptManagerRunning = source["scriptManagerRunning"];
	        this.pythonRuntime = source["pythonRuntime"];
	        this.latestTransferErrors = source["latestTransferErrors"];
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
	export class ScriptInfo {
	    id: string;
	    name: string;
	    description: string;
	    author: string;
	    version: string;
	    size: number;
	    created: string;
	    updated: string;
	    status: string;
	    isInstalled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ScriptInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.author = source["author"];
	        this.version = source["version"];
	        this.size = source["size"];
	        this.created = source["created"];
	        this.updated = source["updated"];
	        this.status = source["status"];
	        this.isInstalled = source["isInstalled"];
	    }
	}

}

