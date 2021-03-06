param(
    $json="",
    $projectdoc="C:\rps-offline\rps-projects.docx",
    $backupwindow=8,
    $wanaccelerator = 3
)

New-Item -Path "c:\" -Name "rps-offline" -ItemType "directory" -ErrorAction Ignore

if (($json -eq "") -or (-not [System.IO.File]::Exists($json))) {
    write-host "Not running without a json file, got|$json|"
} else {
    write-host "Generating doc"
    $log = "c:\rps-offline\log.txt"

    $input = get-content $json | convertfrom-json

    $word = New-Object -ComObject word.application
    $word.Visible = $true

    $doc = $null
    #borrowed code https://techblog.dorogin.com/generate-word-documents-with-powershell-cda654b9cb0e
    if(-not [System.IO.File]::Exists($projectdoc)) {
        $doc = $word.documents.add()
        $doc.SaveAs($projectdoc)
        #"saving new" | out-file -append $log
    } else {
        $doc = $word.Documents.Open($projectdoc)
        
        $wdStory = 6
        $wdMove = 0
        $doc.ActiveWindow.Selection.EndKey($wdStory, $wdMove)
        #"opening old" | out-file -append $log
    }


    $title = (Get-Culture).TextInfo.ToTitleCase([io.path]::GetFileNameWithoutExtension($input.origfname))
    $generated = $input.datestamp
    $url = $input.srcurl
    #$title | out-file -append $log

    function GoToEndOfDoc {
        param(
            $doc
        )
        $wdStory = 6
        $wdMove = 0
        $doc.ActiveWindow.Selection.EndKey($wdStory, $wdMove) | out-null
    }
    function WriteEasy {
        param(
            $doc,
            $texttowrite,
            $style="Normal"
        )
        GoToEndOfDoc -doc $doc
        $sel = $doc.ActiveWindow.Selection
        $sel.Style=$style
        $sel.TypeText($texttowrite)
        $sel.TypeParagraph()
    }
    function WriteEasyLink {
        param(
            $doc,
            $texttowrite,
            $link,
            $style="Normal"
        )
        GoToEndOfDoc -doc $doc
        $sel = $doc.ActiveWindow.Selection
        $sel.Style=$style
        $sel.TypeText($texttowrite)
        $sel.Hyperlinks.Add($sel.Range,$link)
        $sel.TypeParagraph()
    }

    WriteEasy -doc $doc -texttowrite $title -style "Heading 1"

    WriteEasy -doc $doc -texttowrite ""


    $allfiles = $input.files

    <#

    $tab = $doc.ActiveWindow.Selection.Tables.Add($range,,3)
    #>

    $rows = ($allfiles.count)
    $cols = 3

    $tab = $doc.Tables.Add($doc.ActiveWindow.Selection.Range,$rows+5,$cols)
    
    function celltext {
        param(
            $tab,$row,$ret,$fname,$size,$bgcolor=$null
        )
        $tab.Cell($row,1).Range.Text = $ret
        $tab.Cell($row,2).Range.Text = $fname
        $tab.Cell($row,3).Range.Text = $size
        if($bgcolor -ne $null) {
            #RGB = red + (green * 256) + (blue * 65536)
            $rgb = $($bgcolor[0]+$bgcolor[1]*256+$bgcolor[2]*65536)
            write-host "Setting $rgb color"
            $tab.Cell($row,3).range.shading.BackgroundPatternColor = $rgb
        }
    }
    function tbstring {
        param($size)
        return ("{0,2:N}" -f ($size/[math]::Pow(1024,4)))
    }
    celltext -tab $tab -row 1 -ret "Retention" -fname "File Name" -size "In TB"

    $i = 2
    $bigffile = 0
    $bigfsource = 0
    $bigifile = 0
    $bigisource = 0
    $compress = 0

    foreach ($vfile in $allfiles) { 
        $typen = "full.vbk";$full=1
        if ($vfile.backuptype -eq "I") {
            $typen = "incremental.vib"
            $full=0
        } elseif ($vfile.backuptype -eq "R") {
            $typen = "incremental.vrb"
            $full=0
        }
        if ($compress -lt $vfile.transferCompression) {
            $compress = $vfile.transferCompression
        }
        if ($full -eq 1) {
            if ($bigffile -lt $vfile.filesize) {
                $bigffile =  $vfile.filesize
            }
            if ($bigfsource -lt $vfile.source) {
                $bigfsource =  $vfile.source
            }
        } else {
            if ($bigifile -lt $vfile.filesize) {
                $bigifile =  $vfile.filesize
            }
            if ($bigisource -lt $vfile.sourcedelta) {
                $bigisource =  $vfile.sourcedelta
            }
        }

        celltext -tab $tab -row $i -ret $vfile.retention -fname $typen -size $(tbstring -size $vfile.filesize)
        write-host $i,$typen
        $i = $i+1
    }

    celltext -tab $tab -row $($rows+2) -ret ""  -fname "Total Files" -size $(tbstring -size $input.size)
    celltext -tab $tab -row $($rows+3) -ret ""  -fname "Workspace" -size $(tbstring -size $input.workspace)
    celltext -tab $tab -row $($rows+4) -ret ""  -fname "Total Storage" -size $(tbstring -size $input.sizeandworkspace) -bgcolor @(204, 255, 204)

    

    #https://mypowershell.webnode.sk/news/create-word-document-with-multiple-tables-from-powershell/
    #seems to help against corrupted tables (select to end and make a type parageraph)
    $doc.ActiveWindow.Selection.EndKey(6)
    $doc.ActiveWindow.Selection.TypeParagraph()


    WriteEasy -doc $doc -texttowrite ""
    $secwindow = $backupwindow *3600
    WriteEasy -doc $doc -texttowrite ("Backup Window {0} hours x 3600 secs = {1}" -f $backupwindow,$secwindow)
    $bigfmb = $bigfsource/[math]::Pow(1024,2)
    $bigfmbwin = $bigfmb/$secwindow
    $bigfslots = [Math]::Ceiling($bigfmbwin/100)
    WriteEasy -doc $doc -texttowrite ""
    WriteEasy -doc $doc -texttowrite ("Biggest Full Source in TB {0}" -f (tbstring -size $bigfsource))
    WriteEasy -doc $doc -texttowrite ("{0,2:N} MB / {1:D} = {2,2:N}" -f $bigfmb,$secwindow,$bigfmbwin)
    WriteEasy -doc $doc -texttowrite ("{0,2:N} MB / 100 = {1} slots" -f $bigfmb,$bigfslots)

  
    $bigimb = $bigisource/[math]::Pow(1024,2)
    $bigimbwin = $bigimb/$secwindow
    $bigislots = [Math]::Ceiling($bigimbwin/25)
    WriteEasy -doc $doc -texttowrite ""
    WriteEasy -doc $doc -texttowrite ("Biggest Inc Source in TB {0}" -f (tbstring -size $bigisource))
    WriteEasy -doc $doc -texttowrite ("{0,2:N} MB / {1:D} = {2,2:N}" -f $bigimb,$secwindow,$bigimbwin)
    WriteEasy -doc $doc -texttowrite ("{0,2:N} MB / 25 = {1} slots" -f $bigimb,$bigislots)
    WriteEasy -doc $doc -texttowrite ""
    WriteEasyLink -doc $doc -texttowrite "More Information : " -link "https://bp.veeam.expert/proxy_servers_intro/proxy_server_vmware-vsphere/sizing_a_backup_proxy"
    WriteEasy -doc $doc -texttowrite ""

    $bigfproxytorepombcom = $bigfsource*$compress/100/[math]::Pow(1024,2)
    $bigfptrmbs = $bigfproxytorepombcom/$secwindow
    WriteEasy -doc $doc -texttowrite ("Biggest Full Source Compressed in MB {0}" -f $bigfproxytorepombcom)
    WriteEasy -doc $doc -texttowrite ("{0,2:N} MB / {1:D} = {2,2:N} MB/s = {3,2:N} MBit/s" -f $bigfproxytorepombcom,$secwindow,$bigfptrmbs,$($bigfptrmbs*8))
    WriteEasy -doc $doc -texttowrite ""

    $bigiproxytorepombcom = $bigisource*$compress/100/[math]::Pow(1024,2)
    $bigiptrmbs = $bigiproxytorepombcom/$secwindow
    WriteEasy -doc $doc -texttowrite ("Biggest Inc Source Compressed in MB {0}" -f $bigiproxytorepombcom)
    WriteEasy -doc $doc -texttowrite ("{0,2:N} MB / {1:D} = {2,2:N} MB/s = {3,2:N} MBit/s" -f $bigiproxytorepombcom,$secwindow,$bigiptrmbs,$($bigiptrmbs*8))
    WriteEasy -doc $doc -texttowrite ""

    
    WriteEasy -doc $doc -texttowrite ("WAN Accelerator with initial seeding and {0}x improvement on top of compression" -f $wanaccelerator)
    WriteEasy -doc $doc -texttowrite ("{0,2:N} MBit/s / {1} = {2,2:N} MBit/s" -f $($bigiptrmbs*8),$wanaccelerator,$($bigiptrmbs*8/$wanaccelerator))

    WriteEasy -doc $doc -texttowrite ""
    WriteEasy -doc $doc -texttowrite "Generated on $generated"
    WriteEasyLink -doc $doc -texttowrite "Source : " -link $url


    $doc.Save()
    $doc.Close()
    $word.Quit()

    #scary stuff from https://blogs.technet.microsoft.com/heyscriptingguy/2015/02/20/use-powershell-to-add-table-to-word-doc-and-email-as-attachment/
    [System.Runtime.Interopservices.Marshal]::ReleaseComObject($doc) | Out-Null
    [System.Runtime.Interopservices.Marshal]::ReleaseComObject($word) | Out-Null
    [System.Runtime.Interopservices.Marshal]::ReleaseComObject($tab) | Out-Null
    Remove-Variable Doc,Word, Tab
    [gc]::collect()
    [gc]::WaitForPendingFinalizers()
}