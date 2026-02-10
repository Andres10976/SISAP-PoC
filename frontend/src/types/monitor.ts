export interface MonitorStatus {
  is_running: boolean;
  last_run_at: string | null;
  last_tree_size: number;
  last_processed_index: number;
  total_processed: number;
  certs_in_last_cycle: number;
  matches_in_last_cycle: number;
  parse_errors_in_last_cycle: number;
  last_error: string;
  updated_at: string;
}
