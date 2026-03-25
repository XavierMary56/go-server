# Admin Static Files

`internal/admin/static/` is the embedded web UI for the admin console.

## Structure

- [index.html](/d:/Users/Public/php20250819/2026www/go-server/internal/admin/static/index.html)
  The page skeleton and modal markup. It should stay focused on HTML structure.
- [admin-ui.css](/d:/Users/Public/php20250819/2026www/go-server/internal/admin/static/css/admin-ui.css)
  Shared styles for the admin console.
- [admin-common.js](/d:/Users/Public/php20250819/2026www/go-server/internal/admin/static/js/admin-common.js)
  Shared helpers, login flow, tab switching, global state, and delete confirmation.
- [admin-project-keys.js](/d:/Users/Public/php20250819/2026www/go-server/internal/admin/static/js/admin-project-keys.js)
  Project key list, add/edit/view actions, and key dialog behavior.
- [admin-providers-models.js](/d:/Users/Public/php20250819/2026www/go-server/internal/admin/static/js/admin-providers-models.js)
  Anthropic/OpenAI/Grok key management and model management.
- [admin-stats-logs.js](/d:/Users/Public/php20250819/2026www/go-server/internal/admin/static/js/admin-stats-logs.js)
  Project statistics, audit log filters, and log detail dialog.

## Editing Notes

- Keep new page-specific logic in the matching module instead of moving it back into `index.html`.
- Put shared helpers in `admin-common.js` only when they are reused across multiple tabs.
- If a tab grows large again, prefer splitting that tab into another `admin-*.js` file.
- Static files are embedded by [web.go](/d:/Users/Public/php20250819/2026www/go-server/internal/admin/web.go), so any rename needs the HTML references updated and a rebuild to verify.
