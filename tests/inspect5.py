import openpyxl

f = 'D:/Users/Public/php20250819/2026www/go-server/tests/two_site_multilang_rule_test_report_split_20260324_with_translation.xlsx'
wb = openpyxl.load_workbook(f, read_only=True)
ws = wb['dx017_\u03b4\u03c8\u00ae\u00ae']
for row in ws.iter_rows(min_row=2, max_row=5, values_only=True):
    raw = str(row[2]) if row[2] else ''
    print('raw:', repr(raw))
    print('chars:', [hex(ord(c)) for c in raw[:20]])
    print()
wb.close()
