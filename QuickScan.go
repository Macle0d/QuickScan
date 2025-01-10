package main

/*
 Autor: Omar Peña
 Descripción: Herramienta de escaneo rápido de puertos en Go. Inspirado en el FastTcpScan de @s4vitar.
 Repo: https://github.com/Macle0d/QuickScan
 Version: 1.0
*/

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	bold   = "\033[1m"
)

// Parámetros de entrada
var (
	host     = flag.String("host", "127.0.0.1", "IP/dominio o CIDR (p.e. 192.168.1.1/24)")
	ports    = flag.String("range", "1-65535", "Rango de puertos: 80,443,1-65535,1000-2000,...")
	threads  = flag.Int("threads", 1000, "Número de hilos a usar")
	timeout  = flag.Duration("timeout", 1*time.Second, "Segundos por puerto")
	filePath = flag.String("file", "", "Archivo con lista de IP/dominio/CIDR por línea")
)

// Banner
func printBanner() {
	fmt.Println(bold + blue + `
 ╔═╗ ┬ ┬┬┌─┐┬┌─  ╔═╗┌─┐┌─┐┌┐┌
 ║═╬╗│ │││  ├┴┐  ╚═╗│  ├─┤│││
 ╚═╝╚└─┘┴└─┘┴ ┴  ╚═╝└─┘┴ ┴┘└┘` + reset)
	fmt.Println(bold + yellow + "   by: Omar Peña - @p3nt3ster\n" + reset)
}

// expandCIDR convierte una notación CIDR (e.g., "192.168.1.0/24") en una lista de IPs
func expandCIDR(cidr string) []string {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		// Si no es un CIDR válido, retorna vacío
		return []string{}
	}
	var ips []string
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); incIP(ip) {
		ips = append(ips, ip.String())
	}
	return ips
}

// incIP incrementa una IP para iterar sobre el rango del CIDR
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}
}

// parseTargets determina si la entrada es un CIDR o un host individual.
// Retorna la lista de hosts/IPs finales a escanear.
func parseTargets(target string) []string {
	if strings.Contains(target, "/") {
		// Se asume que es CIDR; expandimos a lista de IPs
		expanded := expandCIDR(target)
		// Si el parseo falló, expandCIDR retorna slice vacío; 
		// para ese caso, tratamos el target como un host normal
		if len(expanded) > 0 {
			return expanded
		}
	}
	// No es CIDR o falló parseo => se trata como host único
	return []string{target}
}

// Procesa cadenas de rango, emite puertos individualmente
func processRange(ctx context.Context, r string) chan int {
	c := make(chan int)
	done := ctx.Done()

	go func() {
		defer close(c)
		blocks := strings.Split(r, ",")
		for _, block := range blocks {
			rg := strings.Split(block, "-")
			var minPort, maxPort int
			var err error

			minPort, err = strconv.Atoi(rg[0])
			if err != nil {
				log.Printf(red+"[!] Error interpretando rango: %s"+reset, block)
				continue
			}
			if len(rg) == 1 {
				maxPort = minPort
			} else {
				maxPort, err = strconv.Atoi(rg[1])
				if err != nil {
					log.Printf(red+"[!] Error interpretando rango: %s"+reset, block)
					continue
				}
			}
			for port := minPort; port <= maxPort; port++ {
				select {
				case c <- port:
				case <-done:
					return
				}
			}
		}
	}()
	return c
}

// Inicia hilos de escaneo de puertos
func scanPorts(ctx context.Context, in <-chan int, host string, tOut time.Duration) chan string {
	out := make(chan string)
	done := ctx.Done()
	var wg sync.WaitGroup

	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case port, ok := <-in:
					if !ok {
						return
					}
					s := scanPort(host, port, tOut)
					select {
					case out <- s:
					case <-done:
						return
					}
				case <-done:
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// Escanea un puerto específico
func scanPort(host string, port int, tOut time.Duration) string {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, tOut)
	if err != nil {
		return fmt.Sprintf("%d: Cerrado", port)
	}
	_ = conn.Close()
	return fmt.Sprintf("%d: %sAbierto%s", port, green, reset)
}

// Escanea un host/ IP
func scanHost(ctx context.Context, host string, ports string, tOut time.Duration) {
	fmt.Printf(bold+cyan+"[*] Escaneando host: %s\n"+reset, host)
	fmt.Printf(bold+cyan+"[*] Rango de puertos: %s\n\n"+reset, ports)

	pR := processRange(ctx, ports)
	sP := scanPorts(ctx, pR, host, tOut)

	for port := range sP {
		if strings.Contains(port, "Abierto") {
			fmt.Println(port)
		}
	}
	fmt.Println(bold + yellow + "\n[+] Escaneo completado.\n" + reset)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	flag.Parse()
	printBanner()

	var targets []string

	// Si se especifica -file, leemos línea por línea (cada línea puede ser host o CIDR)
	if *filePath != "" {
		f, err := os.Open(*filePath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			// parseTargets se encarga de ver si es CIDR o single host
			expanded := parseTargets(line)
			targets = append(targets, expanded...)
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	} else {
		// Escaneo único (parámetro -host). Puede ser CIDR o single host
		targets = parseTargets(*host)
	}

	// Escanear cada host resultante
	for _, t := range targets {
		scanHost(ctx, t, *ports, *timeout)
	}
}
