export namespace backup {
	
	export class BackupResult {
	    path: string;
	    fileCount: number;
	
	    static createFrom(source: any = {}) {
	        return new BackupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.fileCount = source["fileCount"];
	    }
	}
	export class FileEntry {
	    name: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new FileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size = source["size"];
	    }
	}
	export class RestoreResult {
	    restored: number;
	
	    static createFrom(source: any = {}) {
	        return new RestoreResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.restored = source["restored"];
	    }
	}

}

export namespace main {
	
	export class BackupInfoView {
	    version: string;
	    createdAt: string;
	    hostname: string;
	    files: backup.FileEntry[];
	    fileCount: number;
	
	    static createFrom(source: any = {}) {
	        return new BackupInfoView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.createdAt = source["createdAt"];
	        this.hostname = source["hostname"];
	        this.files = this.convertValues(source["files"], backup.FileEntry);
	        this.fileCount = source["fileCount"];
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
	export class StatusInfo {
	    os: string;
	    dataDir: string;
	    termiusRunning: boolean;
	    dataDirExists: boolean;
	    dataDirReadable: boolean;
	    dataDirHint: string;
	    closeHint: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.dataDir = source["dataDir"];
	        this.termiusRunning = source["termiusRunning"];
	        this.dataDirExists = source["dataDirExists"];
	        this.dataDirReadable = source["dataDirReadable"];
	        this.dataDirHint = source["dataDirHint"];
	        this.closeHint = source["closeHint"];
	        this.version = source["version"];
	    }
	}

}

