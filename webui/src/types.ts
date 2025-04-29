export interface DownloaderConfig {
  type: "aria2c" | "transmission";
  host?: string;
  port?: number | null;
  rpcPath?: string;
  token?: string;
  username?: string;
  password?: string;
  useHttps?: boolean;
  autoCleanUp?: boolean;
}

export interface FilterConfig {
  include?: string[];
  exclude?: string[];
}

export interface ExtracterConfig {
  tag?: "title" | "link" | "description" | "enclosure" | "guid";
  pattern?: string;
}