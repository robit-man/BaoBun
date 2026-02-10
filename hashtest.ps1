# Parameters
$chunkSize = 64KB
$source = "C:\Users\mitch\Repositories\BaoBun\testclient\test.mp3"
$target = "C:\Users\mitch\Repositories\BaoBun\cmd\client\test.mp3"

$fs1 = [System.IO.File]::OpenRead($source)
$fs2 = [System.IO.File]::OpenRead($target)

$chunkIndex = 0
while ($true) {
    $buf1 = New-Object byte[] $chunkSize
    $buf2 = New-Object byte[] $chunkSize

    $read1 = $fs1.Read($buf1, 0, $chunkSize)
    $read2 = $fs2.Read($buf2, 0, $chunkSize)

    if ($read1 -eq 0 -and $read2 -eq 0) { break }

    if ($read1 -ne $read2 -or !($buf1[0..($read1-1)] -eq $buf2[0..($read2-1)])) {
        Write-Host "Mismatch in chunk $chunkIndex"
    }

    $chunkIndex++
}

$fs1.Close()
$fs2.Close()