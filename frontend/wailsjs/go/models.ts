export namespace main {
	
	export class PriceData {
	    btc: string;
	    eth: string;
	    lastUpdate: string;
	
	    static createFrom(source: any = {}) {
	        return new PriceData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.btc = source["btc"];
	        this.eth = source["eth"];
	        this.lastUpdate = source["lastUpdate"];
	    }
	}

}

