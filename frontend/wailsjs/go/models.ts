export namespace data {
	
	export class DataRequest {
	    type: string;
	    symbol: string;
	    // Go type: time
	    start_date: any;
	    // Go type: time
	    end_date: any;
	    granularity: string;
	
	    static createFrom(source: any = {}) {
	        return new DataRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.symbol = source["symbol"];
	        this.start_date = this.convertValues(source["start_date"], null);
	        this.end_date = this.convertValues(source["end_date"], null);
	        this.granularity = source["granularity"];
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
	    metadata: {[key: string]: string};
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

}

export namespace main {
	
	export class Peer {
	    id: string;
	    address: string;
	    reputation: number;
	    isConnected: boolean;
	    lastSeen: string;
	    roles: string[];
	
	    static createFrom(source: any = {}) {
	        return new Peer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.address = source["address"];
	        this.reputation = source["reputation"];
	        this.isConnected = source["isConnected"];
	        this.lastSeen = source["lastSeen"];
	        this.roles = source["roles"];
	    }
	}
	export class ScriptUploadData {
	    name: string;
	    dataType: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new ScriptUploadData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.dataType = source["dataType"];
	        this.content = source["content"];
	    }
	}

}

export namespace scripts {
	
	export class ScriptExecutor {
	
	
	    static createFrom(source: any = {}) {
	        return new ScriptExecutor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

