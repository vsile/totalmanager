## Инструкция по установке
Totalmanager обеспечивает возможность "прикреплять" локальные файлы к заметкам. Для этого нужно:
- сделать файл changeArg исполняемым
```
    sudo chmod +x /path/to/changeArg
```
- переместить файл local.desktop в директорию $HOME/.local/share/application

- добавить в файл mimeapp.list соответствующие хандлер-схемы:
```
    [Default Applications]
    x-scheme-handler/pdf=local.desktop
    x-scheme-handler/odt=local.desktop
    x-scheme-handler/ocx=local.desktop
    x-scheme-handler/doc=local.desktop
    x-scheme-handler/ods=local.desktop
    x-scheme-handler/xls=local.desktop
    x-scheme-handler/lsx=local.desktop
    x-scheme-handler/jpg=local.desktop
    x-scheme-handler/peg=local.desktop
    x-scheme-handler/png=local.desktop
    x-scheme-handler/tif=local.desktop
```

### Пример файла-сервиса systemd

```
[Unit]
Description=Total Manager v1.0

[Service]
Type=simple
Restart=always
RestartSec=3

Environment=GOPATH=/home/vera/go/
Environment=HOME=/home/vera/
#Environment=GOCACHE=/home/vera/.cache/go-build
WorkingDirectory=/home/vera/go/totalmanager
ExecStart=/home/vera/go/totalmanager/bin/totalmanager

[Install]
WantedBy=multi-user.target
```
