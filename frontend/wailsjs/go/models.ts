export namespace app {
	
	export class Bootstrap {
	    version: string;
	    commit: string;
	    buildDate: string;
	    locale: string;
	    theme: string;
	    visualEffectsPref: string;
	    sidebarCollapsed: boolean;
	    catalogView: string;
	    capabilities: platform.Capabilities;
	    libraryRoot: string;
	    paths: config.Paths;
	    searchIndexReady: boolean;
	    databaseUpdating: boolean;
	    catalogOpening: boolean;
	    catalogReady: boolean;
	    mediaBase?: string;
	    storage: storage.Snapshot;
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
	        this.sidebarCollapsed = source["sidebarCollapsed"];
	        this.catalogView = source["catalogView"];
	        this.capabilities = this.convertValues(source["capabilities"], platform.Capabilities);
	        this.libraryRoot = source["libraryRoot"];
	        this.paths = this.convertValues(source["paths"], config.Paths);
	        this.searchIndexReady = source["searchIndexReady"];
	        this.databaseUpdating = source["databaseUpdating"];
	        this.catalogOpening = source["catalogOpening"];
	        this.catalogReady = source["catalogReady"];
	        this.mediaBase = source["mediaBase"];
	        this.storage = this.convertValues(source["storage"], storage.Snapshot);
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
	export class INPXFile {
	    path: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new INPXFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	    }
	}
	export class ImportPreview {
	    libraryRoot: string;
	    zipCount: number;
	    inpxFiles: INPXFile[];
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
	        this.zipCount = source["zipCount"];
	        this.inpxFiles = this.convertValues(source["inpxFiles"], INPXFile);
	        this.inpxPath = source["inpxPath"];
	        this.inpxFileName = source["inpxFileName"];
	        this.fileVersion = source["fileVersion"];
	        this.catalogVersion = source["catalogVersion"];
	        this.sameVersion = source["sameVersion"];
	        this.hasCatalog = source["hasCatalog"];
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

export namespace catalog {
	
	export class Author {
	    id: number;
	    displayName: string;
	    sortName: string;
	    workCount: number;
	
	    static createFrom(source: any = {}) {
	        return new Author(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.displayName = source["displayName"];
	        this.sortName = source["sortName"];
	        this.workCount = source["workCount"];
	    }
	}
	export class Total {
	    n: number;
	    capped?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Total(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.n = source["n"];
	        this.capped = source["capped"];
	    }
	}
	export class AuthorPage {
	    items: Author[];
	    nextCursor?: string;
	    total?: Total;
	
	    static createFrom(source: any = {}) {
	        return new AuthorPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Author);
	        this.nextCursor = source["nextCursor"];
	        this.total = this.convertValues(source["total"], Total);
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
	export class Genre {
	    id: number;
	    code: string;
	    nameRu: string;
	    workCount: number;
	
	    static createFrom(source: any = {}) {
	        return new Genre(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.code = source["code"];
	        this.nameRu = source["nameRu"];
	        this.workCount = source["workCount"];
	    }
	}
	export class Series {
	    id: number;
	    name: string;
	    sortName: string;
	    workCount: number;
	
	    static createFrom(source: any = {}) {
	        return new Series(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sortName = source["sortName"];
	        this.workCount = source["workCount"];
	    }
	}
	export class Work {
	    id: number;
	    workKey: string;
	    title: string;
	    sortTitle: string;
	    authorsText: string;
	    lang?: string;
	    rating?: number;
	    wantToRead?: boolean;
	    addedDate?: string;
	    series?: string;
	    seriesNo?: string;
	    editionCount: number;
	    hasFile: boolean;
	    size?: number;
	    librate?: number;
	
	    static createFrom(source: any = {}) {
	        return new Work(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workKey = source["workKey"];
	        this.title = source["title"];
	        this.sortTitle = source["sortTitle"];
	        this.authorsText = source["authorsText"];
	        this.lang = source["lang"];
	        this.rating = source["rating"];
	        this.wantToRead = source["wantToRead"];
	        this.addedDate = source["addedDate"];
	        this.series = source["series"];
	        this.seriesNo = source["seriesNo"];
	        this.editionCount = source["editionCount"];
	        this.hasFile = source["hasFile"];
	        this.size = source["size"];
	        this.librate = source["librate"];
	    }
	}
	export class HomeDashboard {
	    worksListable: number;
	    authorsTotal: number;
	    seriesTotal: number;
	    inpxVersion?: string;
	    importedAt?: string;
	    hero?: Work;
	    heroSource?: string;
	    arrivals?: Work[];
	    rated?: Work[];
	    wantToReadCount: number;
	    popularGenres?: Genre[];
	    popularSeries?: Series[];
	
	    static createFrom(source: any = {}) {
	        return new HomeDashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.worksListable = source["worksListable"];
	        this.authorsTotal = source["authorsTotal"];
	        this.seriesTotal = source["seriesTotal"];
	        this.inpxVersion = source["inpxVersion"];
	        this.importedAt = source["importedAt"];
	        this.hero = this.convertValues(source["hero"], Work);
	        this.heroSource = source["heroSource"];
	        this.arrivals = this.convertValues(source["arrivals"], Work);
	        this.rated = this.convertValues(source["rated"], Work);
	        this.wantToReadCount = source["wantToReadCount"];
	        this.popularGenres = this.convertValues(source["popularGenres"], Genre);
	        this.popularSeries = this.convertValues(source["popularSeries"], Series);
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
	export class ListPeopleQuery {
	    letter: string;
	    query: string;
	    cursor: string;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new ListPeopleQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.letter = source["letter"];
	        this.query = source["query"];
	        this.cursor = source["cursor"];
	        this.limit = source["limit"];
	    }
	}
	export class ListWorksQuery {
	    sort: string;
	    cursor: string;
	    lang: string;
	    genreId: number;
	    authorId: number;
	    seriesId: number;
	    rated: boolean;
	    want: boolean;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new ListWorksQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sort = source["sort"];
	        this.cursor = source["cursor"];
	        this.lang = source["lang"];
	        this.genreId = source["genreId"];
	        this.authorId = source["authorId"];
	        this.seriesId = source["seriesId"];
	        this.rated = source["rated"];
	        this.want = source["want"];
	        this.limit = source["limit"];
	    }
	}
	export class SearchQuery {
	    q: string;
	    lang: string;
	    genreId: number;
	    authorId: number;
	    seriesId: number;
	    rated: boolean;
	    want: boolean;
	    offset: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.q = source["q"];
	        this.lang = source["lang"];
	        this.genreId = source["genreId"];
	        this.authorId = source["authorId"];
	        this.seriesId = source["seriesId"];
	        this.rated = source["rated"];
	        this.want = source["want"];
	        this.offset = source["offset"];
	        this.limit = source["limit"];
	    }
	}
	export class WorkPage {
	    items: Work[];
	    nextCursor?: string;
	    total?: Total;
	
	    static createFrom(source: any = {}) {
	        return new WorkPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Work);
	        this.nextCursor = source["nextCursor"];
	        this.total = this.convertValues(source["total"], Total);
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
	export class SearchResult {
	    authors?: Author[];
	    series?: Series[];
	    authorsTotal?: Total;
	    seriesTotal?: Total;
	    works: WorkPage;
	    fallback?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authors = this.convertValues(source["authors"], Author);
	        this.series = this.convertValues(source["series"], Series);
	        this.authorsTotal = this.convertValues(source["authorsTotal"], Total);
	        this.seriesTotal = this.convertValues(source["seriesTotal"], Total);
	        this.works = this.convertValues(source["works"], WorkPage);
	        this.fallback = source["fallback"];
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
	
	export class SeriesPage {
	    items: Series[];
	    nextCursor?: string;
	    total?: Total;
	
	    static createFrom(source: any = {}) {
	        return new SeriesPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], Series);
	        this.nextCursor = source["nextCursor"];
	        this.total = this.convertValues(source["total"], Total);
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
	
	
	export class WorkEdition {
	    id: number;
	    archiveName: string;
	    fileName?: string;
	    fileExt?: string;
	    size?: number;
	    addedDate?: string;
	    preferred?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WorkEdition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.archiveName = source["archiveName"];
	        this.fileName = source["fileName"];
	        this.fileExt = source["fileExt"];
	        this.size = source["size"];
	        this.addedDate = source["addedDate"];
	        this.preferred = source["preferred"];
	    }
	}
	export class WorkDetails {
	    id: number;
	    workKey: string;
	    title: string;
	    sortTitle: string;
	    authorsText: string;
	    lang?: string;
	    rating?: number;
	    wantToRead?: boolean;
	    addedDate?: string;
	    series?: string;
	    seriesNo?: string;
	    editionCount: number;
	    hasFile: boolean;
	    size?: number;
	    librate?: number;
	    comment?: string;
	    authors?: Author[];
	    genres?: Genre[];
	    seriesId?: number;
	    prevWorkId?: number;
	    nextWorkId?: number;
	    annotation?: string;
	    annotationChecked?: boolean;
	    fileExt?: string;
	    preferredEditionId?: number;
	    editions?: WorkEdition[];
	
	    static createFrom(source: any = {}) {
	        return new WorkDetails(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workKey = source["workKey"];
	        this.title = source["title"];
	        this.sortTitle = source["sortTitle"];
	        this.authorsText = source["authorsText"];
	        this.lang = source["lang"];
	        this.rating = source["rating"];
	        this.wantToRead = source["wantToRead"];
	        this.addedDate = source["addedDate"];
	        this.series = source["series"];
	        this.seriesNo = source["seriesNo"];
	        this.editionCount = source["editionCount"];
	        this.hasFile = source["hasFile"];
	        this.size = source["size"];
	        this.librate = source["librate"];
	        this.comment = source["comment"];
	        this.authors = this.convertValues(source["authors"], Author);
	        this.genres = this.convertValues(source["genres"], Genre);
	        this.seriesId = source["seriesId"];
	        this.prevWorkId = source["prevWorkId"];
	        this.nextWorkId = source["nextWorkId"];
	        this.annotation = source["annotation"];
	        this.annotationChecked = source["annotationChecked"];
	        this.fileExt = source["fileExt"];
	        this.preferredEditionId = source["preferredEditionId"];
	        this.editions = this.convertValues(source["editions"], WorkEdition);
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

export namespace covers {
	
	export class Annotation {
	    text?: string;
	    checked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Annotation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.checked = source["checked"];
	    }
	}
	export class Progress {
	    total: number;
	    done: number;
	    running: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Progress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.done = source["done"];
	        this.running = source["running"];
	    }
	}

}

export namespace downloads {
	
	export class Result {
	    path: string;
	    fileName: string;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.fileName = source["fileName"];
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
	    finishedAt?: string;
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
	        this.finishedAt = source["finishedAt"];
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

export namespace personal {
	
	export class ExportResult {
	    path: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.count = source["count"];
	    }
	}
	export class ImportNote {
	    workKey?: string;
	    title?: string;
	    field?: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workKey = source["workKey"];
	        this.title = source["title"];
	        this.field = source["field"];
	        this.reason = source["reason"];
	    }
	}
	export class ImportReport {
	    applied: number;
	    skipped: number;
	    notFound: number;
	    invalid: number;
	    notes?: ImportNote[];
	    notFoundPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.applied = source["applied"];
	        this.skipped = source["skipped"];
	        this.notFound = source["notFound"];
	        this.invalid = source["invalid"];
	        this.notes = this.convertValues(source["notes"], ImportNote);
	        this.notFoundPath = source["notFoundPath"];
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
	export class Snapshot {
	    unsyncedCount: number;
	    lastExportAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unsyncedCount = source["unsyncedCount"];
	        this.lastExportAt = source["lastExportAt"];
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

export namespace storage {
	
	export class DumpOffer {
	    path: string;
	    name: string;
	    fileVersion?: string;
	    catalogVersion?: string;
	
	    static createFrom(source: any = {}) {
	        return new DumpOffer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.fileVersion = source["fileVersion"];
	        this.catalogVersion = source["catalogVersion"];
	    }
	}
	export class Snapshot {
	    configured: boolean;
	    available: boolean;
	    unreachable: boolean;
	    libraryRoot?: string;
	    remapped?: boolean;
	    dumpOffer?: DumpOffer;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configured = source["configured"];
	        this.available = source["available"];
	        this.unreachable = source["unreachable"];
	        this.libraryRoot = source["libraryRoot"];
	        this.remapped = source["remapped"];
	        this.dumpOffer = this.convertValues(source["dumpOffer"], DumpOffer);
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

