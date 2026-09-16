// Package network fournit des utilitaires pour récupérer les informations
// réseau de la machine et afficher les URLs d'accès au serveur.
package network

import (
	"fmt"
	"net"
	"os"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// AccessInfo contient toutes les URLs d'accès au serveur.
type AccessInfo struct {
	IPs  []string // Toutes les IPs locales disponibles
	Port int      // Port HTTP
	Name string   // Nom mDNS (ex: "magenta")
}

// GetLocalIPs retourne toutes les adresses IPv4 locales de la machine,
// en excluant les adresses loopback (127.x.x.x).
func GetLocalIPs() []string {
	var ips []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		// Ignorer les interfaces désactivées et loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// On ne garde que les IPv4 non-loopback
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}

			ips = append(ips, ip.String())
		}
	}

	return ips
}

// PrintBanner affiche un banner de démarrage avec toutes les URLs d'accès
// et un QR code pour accès rapide depuis un téléphone.
func PrintBanner(info AccessInfo) {
	ips := GetLocalIPs()

	border := strings.Repeat("─", 52)

	fmt.Println()
	fmt.Printf("  ┌%s┐\n", border)
	fmt.Printf("  │%s│\n", center("🎬  MAGENTA  🎬", 52))
	fmt.Printf("  ├%s┤\n", border)
	fmt.Printf("  │%s│\n", center("Accès depuis le réseau local", 52))
	fmt.Printf("  ├%s┤\n", border)

	// Affichage de toutes les URLs HTTP
	for _, ip := range ips {
		url := fmt.Sprintf("http://%s:%d", ip, info.Port)
		fmt.Printf("  │  🌐  %-46s│\n", url)
	}

	// Affichage de l'URL mDNS
	if info.Name != "" {
		mdnsURL := fmt.Sprintf("http://%s.local:%d", info.Name, info.Port)
		fmt.Printf("  │  📡  %-46s│\n", mdnsURL)
	}

	fmt.Printf("  ├%s┤\n", border)
	fmt.Printf("  │%s│\n", center("💡 Scanner le QR code depuis votre téléphone", 52))
	fmt.Printf("  └%s┘\n", border)
	fmt.Println()

	// Génération et affichage du QR code pour la première IP disponible
	if len(ips) > 0 {
		primaryURL := fmt.Sprintf("http://%s:%d", ips[0], info.Port)
		printQRCode(primaryURL)
	}
}

// printQRCode génère et affiche un QR code dans le terminal.
// Utilise des caractères Unicode demi-blocs pour un rendu correct dans tous les terminaux.
func printQRCode(url string) {
	qr, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		fmt.Fprintf(os.Stderr, "QR code indisponible: %v\n", err)
		return
	}

	// Récupération de la bitmap du QR code
	bitmap := qr.Bitmap()
	if len(bitmap) == 0 {
		return
	}

	fmt.Printf("  QR Code → %s\n\n", url)

	// Rendu avec des blocs Unicode (▀ ▄ █ )
	// On traite 2 lignes à la fois pour compacter le rendu
	for y := 0; y < len(bitmap)-1; y += 2 {
		fmt.Print("  ")
		for x := 0; x < len(bitmap[y]); x++ {
			top := bitmap[y][x]
			bot := bitmap[y+1][x]

			switch {
			case top && bot:
				fmt.Print("█") // les deux noirs
			case top && !bot:
				fmt.Print("▀") // haut noir
			case !top && bot:
				fmt.Print("▄") // bas noir
			default:
				fmt.Print(" ") // les deux blancs
			}
		}
		fmt.Println()
	}
	fmt.Println()
}

// center centre une chaîne dans un champ de largeur donnée.
func center(s string, width int) string {
	// Comptage des caractères visibles (les emojis comptent pour 2)
	visLen := visibleLength(s)
	if visLen >= width {
		return s
	}
	pad := (width - visLen) / 2
	return strings.Repeat(" ", pad) + s + strings.Repeat(" ", width-visLen-pad)
}

// visibleLength calcule la largeur visible d'une chaîne dans le terminal.
// Les emojis et certains caractères Unicode prennent 2 colonnes.
func visibleLength(s string) int {
	length := 0
	for _, r := range s {
		if r > 0x2E7F {
			length += 2 // Caractères larges (emojis, CJK, etc.)
		} else {
			length++
		}
	}
	return length
}
