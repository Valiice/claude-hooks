# Claude Code Stop Notification - Simple Version
Add-Type -AssemblyName System.Windows.Forms

# Play sound
[System.Media.SystemSounds]::Exclamation.Play()

# Show popup notification
$notification = New-Object System.Windows.Forms.NotifyIcon
$notification.Icon = [System.Drawing.SystemIcons]::Information
$notification.Visible = $true
$notification.BalloonTipTitle = "Claude Code"
$notification.BalloonTipText = "Task completed!"
$notification.ShowBalloonTip(5000)

# Keep script alive long enough to show notification
Start-Sleep -Seconds 1
$notification.Dispose()
