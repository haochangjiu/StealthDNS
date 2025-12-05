package cert

import (
	"crypto"
	"crypto/x509"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
)

const rootName = "rootCA.pem"
const rootKeyName = "rootCA-key.pem"

type mkcert struct {
	installMode, uninstallMode bool
	pkcs12, ecdsa, client      bool
	keyFile, certFile, p12File string
	csrPath                    string

	CAROOT string
	caCert *x509.Certificate
	caKey  crypto.PrivateKey

	ignoreCheckFailure bool
}

func Install(ensureFile bool) error {
	m := &mkcert{}
	m.CAROOT = getCAROOT()
	if m.CAROOT == "" {
		log.Fatalln("ERROR: failed to find the default CA location, set one as the CAROOT env var")
	}
	fatalIfErr(os.MkdirAll(m.CAROOT, 0755), "failed to create the CAROOT")
	err := m.loadCA(ensureFile)
	if err != nil {
		return err
	}

	m.install()
	return nil
}

func Uninstall() error {
	m := &mkcert{}
	m.CAROOT = getCAROOT()
	if m.CAROOT == "" {
		log.Fatalln("ERROR: failed to find the default CA location, set one as the CAROOT env var")
	}
	fatalIfErr(os.MkdirAll(m.CAROOT, 0755), "failed to create the CAROOT")
	err := m.loadCA(false)
	if err != nil {
		return err
	}
	m.uninstall()
	return nil
}

func CreateCert(csrFile string, domainName string) error {
	m := &mkcert{}
	m.CAROOT = getCAROOT()
	if m.CAROOT == "" {
		log.Fatalln("ERROR: failed to find the default CA location, set one as the CAROOT env var")
	}
	fatalIfErr(os.MkdirAll(m.CAROOT, 0755), "failed to create the CAROOT")
	err := m.loadCA(false)
	if err != nil {
		return err
	}
	if len(csrFile) > 0 {
		m.csrPath = csrFile
		m.makeCertFromCSR()
		return nil
	}

	if len(domainName) == 0 {
		return errors.New("csr-file and domain-name cannot both be empty")
	}
	return m.create(strings.Split(domainName, " "))
}

func (m *mkcert) create(args []string) error {
	var warning bool
	if storeEnabled("system") && !m.checkPlatform() {
		warning = true
		log.Println("Note: the local CA is not installed in the system trust store.")
	}
	if storeEnabled("nss") && hasNSS && CertutilInstallHelp != "" && !m.checkNSS() {
		warning = true
		log.Printf("Note: the local CA is not installed in the %s trust store.", NSSBrowsers)
	}
	if warning {
		log.Println("Run \"stealth-dns install-root-ca\" for certificates to be trusted automatically")
	}
	m.makeCert(args)
	return nil
}

func getCAROOT() string {
	exeFilePath, err := os.Executable()
	if err != nil {
		return ""
	}
	exeDirPath := filepath.Dir(exeFilePath)
	return filepath.Join(exeDirPath, "etc", "cert")
}

func (m *mkcert) install() {
	if storeEnabled("system") {
		if m.checkPlatform() {
			log.Print("The local CA is already installed in the system trust store!")
		} else {
			if m.installPlatform() {
				log.Print("The local CA is now installed in the system trust store!")
			}
			m.ignoreCheckFailure = true
		}
	}
	if storeEnabled("nss") && hasNSS {
		if m.checkNSS() {
			log.Printf("The local CA is already installed in the %s trust store!", NSSBrowsers)
		} else {
			if hasCertutil && m.installNSS() {
				log.Printf("The local CA is now installed in the %s trust store (requires browser restart)!", NSSBrowsers)
			} else if CertutilInstallHelp == "" {
				log.Printf(`Note: %s support is not available on your platform.`, NSSBrowsers)
			} else if !hasCertutil {
				log.Printf(`Warning: "certutil" is not available, so the CA can't be automatically installed in %s!`, NSSBrowsers)
				log.Printf(`Install "certutil" with "%s" and re-run "stealth-dns install-root-ca"`, CertutilInstallHelp)
			}
		}
	}
	log.Print("")
}

func (m *mkcert) uninstall() {
	if storeEnabled("nss") && hasNSS {
		if hasCertutil {
			m.uninstallNSS()
		} else if CertutilInstallHelp != "" {
			log.Print("")
			log.Printf(`Warning: "certutil" is not available, so the CA can't be automatically uninstalled from %s (if it was ever installed)!`, NSSBrowsers)
			log.Printf(`You can install "certutil" with "%s" and re-run "stealth-dns uninstall-root-ca"`, CertutilInstallHelp)
			log.Print("")
		}
	}
	if storeEnabled("system") && m.uninstallPlatform() {
		log.Print("The local CA is now uninstalled from the system trust store(s)!")
		log.Print("")
	} else if storeEnabled("nss") && hasCertutil {
		log.Printf("The local CA is now uninstalled from the %s trust store(s)!", NSSBrowsers)
		log.Print("")
	}
}

func (m *mkcert) checkPlatform() bool {
	if m.ignoreCheckFailure {
		return true
	}

	_, err := m.caCert.Verify(x509.VerifyOptions{})
	return err == nil
}

func storeEnabled(name string) bool {
	stores := os.Getenv("TRUST_STORES")
	if stores == "" {
		return true
	}
	for _, store := range strings.Split(stores, ",") {
		if store == name {
			return true
		}
	}
	return false
}

func fatalIfErr(err error, msg string) {
	if err != nil {
		log.Fatalf("ERROR: %s: %s", msg, err)
	}
}

func fatalIfCmdErr(err error, cmd string, out []byte) {
	if err != nil {
		log.Fatalf("ERROR: failed to execute \"%s\": %s\n\n%s\n", cmd, err, out)
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func binaryExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

var sudoWarningOnce sync.Once

func commandWithSudo(cmd ...string) *exec.Cmd {
	if u, err := user.Current(); err == nil && u.Uid == "0" {
		return exec.Command(cmd[0], cmd[1:]...)
	}
	if !binaryExists("sudo") {
		sudoWarningOnce.Do(func() {
			log.Println(`Warning: "sudo" is not available, and not running as root. The (un)install operation might fail.`)
		})
		return exec.Command(cmd[0], cmd[1:]...)
	}
	return exec.Command("sudo", append([]string{"--prompt=Sudo password:", "--"}, cmd...)...)
}
