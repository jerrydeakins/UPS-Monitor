#define MyAppName "UPS Monitor"
#ifndef MyAppVersion
  #define MyAppVersion "1.0.0"
#endif
#define MyAppPublisher "AB-Robotron"
#define MyAppExeName "UPS-Monitor.exe"

[Setup]
AppId={{71B099DD-F7A9-463D-BDCE-90BB1BDDD047}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}

DefaultDirName={autopf}\UPS Monitor
DefaultGroupName=UPS Monitor

OutputDir=.
OutputBaseFilename=UPS-Monitor-{#MyAppVersion}-Setup

Compression=lzma
SolidCompression=yes

ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

PrivilegesRequired=admin

UninstallDisplayIcon={app}\{#MyAppExeName}

[Files]
Source: "..\UPS-Monitor.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\UPS Monitor"; Filename: "{app}\{#MyAppExeName}"
Name: "{autodesktop}\UPS Monitor"; Filename: "{app}\{#MyAppExeName}"

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "Запустить UPS Monitor"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: filesandordirs; Name: "{app}"