# QuickScan

Escáner de puertos escrito en Go que permite escanear rangos de puertos a un host específico o a múltiples hosts listados en un archivo.

## Compilación

```bash
go build -ldflags "-s -w" QuickScan.go
```

### Uso
```bash
Usage of ./QuickScan:
  -file string
        Archivo con lista de hosts (IP, dominio, subdominio) por línea
  -host string
        Host o dirección IP a escanear (default "127.0.0.1")
  -range string
        Rango de puertos: 80,443,1-65535,1000-2000,... (default "1-65535")
  -threads int
        Número de hilos a usar (default 1000)
  -timeout duration
        Segundos por puerto (default 1s)
```

## Autor
- Omar Peña - [@Macle0d](https://github.com/Macle0d) - [@p3nt3ster](https://x.com/p3nt3ster)
