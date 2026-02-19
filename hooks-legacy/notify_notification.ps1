# Claude Code Notification - Simple Version
Add-Type -AssemblyName System.Windows.Forms

# Play sound
[System.Media.SystemSounds]::Asterisk.Play()

# Show popup notification
$notification = New-Object System.Windows.Forms.NotifyIcon
$notification.Icon = [System.Drawing.SystemIcons]::Exclamation
$notification.Visible = $true
$notification.BalloonTipTitle = "Claude Code"
$notification.BalloonTipText = "Needs your attention!"
$notification.ShowBalloonTip(5000)

# Keep script alive long enough to show notification
Start-Sleep -Seconds 1
$notification.Dispose()
