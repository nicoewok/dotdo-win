; Inno Setup Script for dotdo Task Manager
; Defines installer configuration, shortcuts, bundled assets, and complete uninstaller cleanup.

#define MyAppName "dotdo"
#ifndef MyAppVersion
#define MyAppVersion "1.0.1"
#endif
#define MyAppPublisher "nicoewok"
#define MyAppURL "https://github.com/nicoewok/dotdo-win"
#define MyAppExeName "dotdo.exe"

[Setup]
; Unique App ID (generated GUID for dotdo)
AppId={{D37E84B1-294F-4C2E-8932-5F32C3A7A9BF}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
LicenseFile=
OutputDir=..\dist
OutputBaseFilename=dotdo-Setup-{#MyAppVersion}
SetupIconFile=..\assets\icon.ico
UninstallDisplayIcon={app}\{#MyAppExeName}
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "..\dotdo.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\assets\*"; DestDir: "{app}\assets"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\assets\icon.ico"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\assets\icon.ico"; Tasks: desktopicon

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: filesandordirs; Name: "{userappdata}\dotdo"
Type: filesandordirs; Name: "{localappdata}\dotdo"
Type: filesandordirs; Name: "{app}"

[Code]
// Ensures complete deletion of application data (%APPDATA%\dotdo and %LOCALAPPDATA%\dotdo) during uninstallation
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  AppDataDir: String;
  LocalAppDataDir: String;
begin
  if CurUninstallStep = usPostUninstall then
  begin
    AppDataDir := ExpandConstant('{userappdata}\dotdo');
    if DirExists(AppDataDir) then
    begin
      DelTree(AppDataDir, True, True, True);
    end;

    LocalAppDataDir := ExpandConstant('{localappdata}\dotdo');
    if DirExists(LocalAppDataDir) then
    begin
      DelTree(LocalAppDataDir, True, True, True);
    end;
  end;
end;

