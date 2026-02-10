import { exportCertificatesUrl } from "../../api/certificates";

export function ExportButton() {
  function handleExport() {
    // Direct download via browser navigation — no fetch needed
    window.open(exportCertificatesUrl(), "_blank");
  }

  return (
    <button
      onClick={handleExport}
      className="rounded-md border border-gray-700 bg-gray-800 px-3 py-2
                 text-sm text-gray-300 hover:bg-gray-700 transition-colors"
    >
      Export CSV
    </button>
  );
}
