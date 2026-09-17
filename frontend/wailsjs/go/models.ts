export namespace app {
	
	export class Bootstrap {
	    version: string;
	    commit: string;
	    buildDate: string;
	    locale: string;
	    theme: string;
	    visualEffectsPref: string;
	    capabilities: platform.Capabilities;
	    libraryRoot: string;
	    paths: config.Paths;
	    startupError?: apperr.Public;
	
	    static createFrom(source: any = {}) {
	        return new Bootstrap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.commit = source["commit"];
	        this.buildDate = source["buildDate"];
	        this.locale = source["locale"];
	        this.theme = source["theme"];
	        this.visualEffectsPref = source["visualEffectsPref"];
	        this.capabilities = this.convertValues(source["capabilities"], platform.Capabilities);
	        this.libraryRoot = source["libraryRoot"];
	        this.paths = this.convertValues(source["paths"], config.Paths);
	        this.startupError = this.convertValues(source["startupError"], apperr.Public);
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

export namespace config {
	
	export class Paths {
	    dataDir: string;
	    dbPath: string;
	    coversDir: string;
	    logsDir: string;
	    backupsDir: string;
	    downloadsDir: string;
	    configPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Paths(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataDir = source["dataDir"];
	        this.dbPath = source["dbPath"];
	        this.coversDir = source["coversDir"];
	        this.logsDir = source["logsDir"];
	        this.backupsDir = source["backupsDir"];
	        this.downloadsDir = source["downloadsDir"];
	        this.configPath = source["configPath"];
	    }
	}

}

export namespace platform {
	
	export class Capabilities {
	    os: string;
	    backdropFilter: boolean;
	    framelessOk: boolean;
	    effectiveEffects: string;
	
	    static createFrom(source: any = {}) {
	        return new Capabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.backdropFilter = source["backdropFilter"];
	        this.framelessOk = source["framelessOk"];
	    this.effectiveEffects = source["effectiveEffects"];
	    }
	}

}

export namespace apperr {
	
	export class Public {
	    code: string;
	    params?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Public(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.params = source["params"];
	    }
	}

}


