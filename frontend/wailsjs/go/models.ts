export namespace config {
	
	export class Session {
	    name: string;
	    host: string;
	    port: number;
	    username: string;
	    private_key?: string;
	    group?: string;
	    last_used: string;
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.private_key = source["private_key"];
	        this.group = source["group"];
	        this.last_used = source["last_used"];
	    }
	}

}

export namespace main {
	
	export class FileItem {
	    name: string;
	    size: string;
	    mode: string;
	    time: string;
	    is_dir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size = source["size"];
	        this.mode = source["mode"];
	        this.time = source["time"];
	        this.is_dir = source["is_dir"];
	    }
	}
	export class TunnelInfo {
	    id: string;
	    local_port: number;
	    remote_host: string;
	    remote_port: number;
	
	    static createFrom(source: any = {}) {
	        return new TunnelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.local_port = source["local_port"];
	        this.remote_host = source["remote_host"];
	        this.remote_port = source["remote_port"];
	    }
	}

}

export namespace transfer {
	
	export class TransferItem {
	    id: number;
	    name: string;
	    remote_path: string;
	    local_path: string;
	    direction: number;
	    total_bytes: number;
	    transfer_bytes: number;
	    status: number;
	    error: string;
	    start_time: string;
	    end_time: string;
	
	    static createFrom(source: any = {}) {
	        return new TransferItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.remote_path = source["remote_path"];
	        this.local_path = source["local_path"];
	        this.direction = source["direction"];
	        this.total_bytes = source["total_bytes"];
	        this.transfer_bytes = source["transfer_bytes"];
	        this.status = source["status"];
	        this.error = source["error"];
	        this.start_time = source["start_time"];
	        this.end_time = source["end_time"];
	    }
	}

}

