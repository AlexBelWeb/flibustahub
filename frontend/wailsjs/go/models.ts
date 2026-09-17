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
	    searchIndexReady: boolean;
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
	        this.searchIndexReady = source["searchIndexReady"];
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
	export class ImportPreview {
	    libraryRoot: string;
	    inpxPath: string;
	    inpxFileName: string;
	    fileVersion: string;
	    catalogVersion: string;
	    sameVersion: boolean;
	    hasCatalog: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ImportPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.libraryRoot = source["libraryRoot"];
	        this.inpxPath = source["inpxPath"];
	        this.inpxFileName = source["inpxFileName"];
	        this.fileVersion = source["fileVersion"];
	        this.catalogVersion = source["catalogVersion"];
	        this.sameVersion = source["sameVersion"];
	        this.hasCatalog = source["hasCatalog"];
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

export namespace inpximport {
	
	export class EncodingsDTO {
	    utf8: number;
	    cp1251: number;
	    versionInfo?: string;
	    collectionInfo?: string;
	
	    static createFrom(source: any = {}) {
	        return new EncodingsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.utf8 = source["utf8"];
	        this.cp1251 = source["cp1251"];
	        this.versionInfo = source["versionInfo"];
	        this.collectionInfo = source["collectionInfo"];
	    }
	}
	export class NotesDTO {
	    missingArchives: string[];
	    missingArchivesTotal: number;
	    unnamedGenres: string[];
	    unnamedGenresTotal: number;
	    skippedMalformed: number;
	    skippedNoLibid: number;
	    encodings: EncodingsDTO;
	    genreNamesMapped: number;
	    phasesMs?: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new NotesDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.missingArchives = source["missingArchives"];
	        this.missingArchivesTotal = source["missingArchivesTotal"];
	        this.unnamedGenres = source["unnamedGenres"];
	        this.unnamedGenresTotal = source["unnamedGenresTotal"];
	        this.skippedMalformed = source["skippedMalformed"];
	        this.skippedNoLibid = source["skippedNoLibid"];
	        this.encodings = this.convertValues(source["encodings"], EncodingsDTO);
	        this.genreNamesMapped = source["genreNamesMapped"];
	        this.phasesMs = source["phasesMs"];
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
	export class ReportDTO {
	    id: number;
	    status: string;
	    inpxPath: string;
	    inpxVersion: string;
	    recordsSeen: number;
	    worksAdded: number;
	    editionsAdded: number;
	    editionsUpdated: number;
	    editionsDeactivated: number;
	    libidCollisions: number;
	    notes: NotesDTO;
	
	    static createFrom(source: any = {}) {
	        return new ReportDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.inpxPath = source["inpxPath"];
	        this.inpxVersion = source["inpxVersion"];
	        this.recordsSeen = source["recordsSeen"];
	        this.worksAdded = source["worksAdded"];
	        this.editionsAdded = source["editionsAdded"];
	        this.editionsUpdated = source["editionsUpdated"];
	        this.editionsDeactivated = source["editionsDeactivated"];
	        this.libidCollisions = source["libidCollisions"];
	        this.notes = this.convertValues(source["notes"], NotesDTO);
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

