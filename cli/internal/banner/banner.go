package banner

import "fmt"

const (
	green = "\033[0;32m"
	dim   = "\033[2m"
	reset = "\033[0m"
)

func Print() {
	fmt.Println()
	fmt.Println(green + "  ██╗      █████╗ ██████╗ ██╗   ██╗██████╗ ██╗███╗   ██╗████████╗██╗  ██╗" + reset)
	fmt.Println(green + "  ██║     ██╔══██╗██╔══██╗╚██╗ ██╔╝██╔══██╗██║████╗  ██║╚══██╔══╝██║  ██║" + reset)
	fmt.Println(green + "  ██║     ███████║██████╔╝ ╚████╔╝ ██████╔╝██║██╔██╗ ██║   ██║   ███████║" + reset)
	fmt.Println(green + "  ██║     ██╔══██║██╔══██╗  ╚██╔╝  ██╔══██╗██║██║╚██╗██║   ██║   ██╔══██║" + reset)
	fmt.Println(green + "  ███████╗██║  ██║██████╔╝   ██║   ██║  ██║██║██║ ╚████║   ██║   ██║  ██║" + reset)
	fmt.Println(green + "  ╚══════╝╚═╝  ╚═╝╚═════╝    ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝   ╚═╝   ╚═╝  ╚═╝" + reset)
	fmt.Println()
	fmt.Println("  " + dim + "Adversarial Cognitive Honeypot Architecture" + reset)
	fmt.Println()
}
