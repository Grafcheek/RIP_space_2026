$InputDocx = "C:\Users\admin\OneDrive\Desktop\Education\RIP_space_2026\Отчёты\ИУ5-65Б Крылов РПЗ.docx"
$OutputTxt = "C:\Users\admin\OneDrive\Desktop\Education\RIP_space_2026\Отчёты\_rpd_export.txt"

$word = New-Object -ComObject Word.Application
$word.Visible = $false
$word.DisplayAlerts = 0

try {
  $doc = $word.Documents.Open($InputDocx, $false, $true)
  # 2 = wdFormatText
  $doc.SaveAs([ref]$OutputTxt, [ref]2)
  $doc.Close()
}
finally {
  $word.Quit()
  if ($doc) { [void][System.Runtime.InteropServices.Marshal]::ReleaseComObject($doc) }
  if ($word) { [void][System.Runtime.InteropServices.Marshal]::ReleaseComObject($word) }
}

Write-Output "OK"

