package main

import (
        "context"
        "flag"
        "fmt"
        "log"
        "net/http"
        "os"
        "os/exec"
        "path/filepath"
        "strings"
        "text/template"
        "time"

        "github.com/google/go-github/v53/github"
)

const (
        serviceTemplate = `[Unit]
Description=Sync GitHub repositories for {{.Username}}
After=network.target

[Service]
Type=oneshot
ExecStart={{.ExecutablePath}} {{.Username}}

[Install]
WantedBy=default.target
`

        timerTemplate = `[Unit]
Description=Run GitHub sync every day at {{.Time}}

[Timer]
OnCalendar={{.Time}}
Persistent=true

[Install]
WantedBy=timers.target
`
        serviceFileName = "github-sync-%s.service"
        timerFileName   = "github-sync-%s.timer"
)

type Config struct {
        Username       string
        ExecutablePath string
        Time           string
}

func main() {
        installFlag := flag.Bool("install", false, "Install systemd --user timer")
        timeFlag := flag.String("t", "03:00", "Time for daily sync (hh:mm)")
        uninstallFlag := flag.Bool("uninstall", false, "Uninstall systemd --user timer")

        flag.Parse()

        if *installFlag {
        if flag.NArg() != 1 {
            fmt.Println("Usage: go run main.go --install <github_username> -t <hh:mm>")
            return
        }
                username := flag.Arg(0)
                installUserSystemdTimer(username, *timeFlag)
                return
        }
        if *uninstallFlag {
        if flag.NArg() != 1 {
            fmt.Println("Usage: go run main.go --uninstall <github_username>")
            return
        }
                username := flag.Arg(0)
                uninstallUserSystemdTimer(username)
                return
        }

    if flag.NArg() != 1 {
        fmt.Println("Usage: go run main.go <github_username>")
        return
    }

    username := flag.Arg(0)
        // Get the directory where the executable is located
        executablePath, err := os.Executable()
        if err != nil {
                log.Fatalf("Error getting executable path: %v", err)
        }
        baseDir := filepath.Dir(executablePath)

        ctx := context.Background()
        client := github.NewClient(http.DefaultClient)

        repos, _, err := client.Repositories.List(ctx, username, nil)
        if err != nil {
                log.Fatalf("Error getting repository list: %v", err)
        }

        for _, repo := range repos {
                repoName := *repo.Name
                repoURL := *repo.CloneURL

                if !isRepoPubliclyAccessible(repoURL) {
                        fmt.Printf("Skipping repository (authentication required): %s\n", colorize(repoName, "red"))
                        continue
                }

                repoPath := filepath.Join(baseDir, repoName)

                if _, err := os.Stat(repoPath); os.IsNotExist(err) {
                        fmt.Printf("Cloning %s...\n", colorize(repoName, "green"))
                        err := cloneRepo(repoURL, repoPath)
                        if err != nil {
                                log.Printf("Error cloning %s: %v\n", colorize(repoName, "red"), err)
                        } else {
                                fmt.Printf("Cloning %s completed\n", colorize(repoName, "green"))
                        }
                } else {
                        fmt.Printf("Synchronizing %s...\n", colorize(repoName, "green"))
                        err := syncRepo(repoPath)
                        if err != nil {
                                log.Printf("Error synchronizing %s: %v\n", colorize(repoName, "red"), err)
                        } else {
                                fmt.Printf("Synchronization of %s completed\n", colorize(repoName, "green"))
                        }
                }
        }

        fmt.Println("Done.")
}

func installUserSystemdTimer(username string, timeString string) {
        executablePath, err := os.Executable()
        if err != nil {
                log.Fatalf("Error getting executable path: %v", err)
        }

        config := Config{
                Username:       username,
                ExecutablePath: executablePath,
                Time:           timeString,
        }

        err = createUserSystemdFiles(config)
        if err != nil {
                log.Fatalf("Error creating user systemd files: %v", err)
        }

        fmt.Println("Systemd --user timer installed successfully.")
        enableAndStartUserTimer(username)

}

func uninstallUserSystemdTimer(username string) {
        err := disableAndStopUserTimer(username)
        if err != nil {
                log.Fatalf("Error disabling and stopping user timer %v", err)
        }
        err = removeUserSystemdFiles(username)
        if err != nil {
                log.Fatalf("Error removing user systemd files %v", err)
        }
        fmt.Println("Systemd --user timer uninstalled successfully.")
}

func createUserSystemdFiles(config Config) error {
        serviceFileName := fmt.Sprintf(serviceFileName, config.Username)
        timerFileName := fmt.Sprintf(timerFileName, config.Username)

    userConfigDir, err := os.UserConfigDir()
    if err != nil {
        return fmt.Errorf("error getting user config dir: %w", err)
    }

    systemdDir := filepath.Join(userConfigDir, "systemd", "user")

    err = os.MkdirAll(systemdDir, 0755)
    if err != nil {
        return fmt.Errorf("error creating systemd dir: %w", err)
    }

        serviceFilePath := filepath.Join(systemdDir, serviceFileName)
        timerFilePath := filepath.Join(systemdDir, timerFileName)

        // Create Service file
        serviceFile, err := os.Create(serviceFilePath)
        if err != nil {
                return fmt.Errorf("error creating service file: %w", err)
        }
        defer serviceFile.Close()

        serviceTmpl := template.Must(template.New("service").Parse(serviceTemplate))
        err = serviceTmpl.Execute(serviceFile, config)
        if err != nil {
                return fmt.Errorf("error executing service template: %w", err)
        }

        // Create Timer file
        timerFile, err := os.Create(timerFilePath)
        if err != nil {
                return fmt.Errorf("error creating timer file: %w", err)
        }
        defer timerFile.Close()

        timerTmpl := template.Must(template.New("timer").Parse(timerTemplate))
        err = timerTmpl.Execute(timerFile, config)
        if err != nil {
                return fmt.Errorf("error executing timer template %w", err)
        }
        return nil
}

func enableAndStartUserTimer(username string) error {
        timerFileName := fmt.Sprintf(timerFileName, username)

        cmd := exec.Command("systemctl", "--user", "enable", timerFileName)
        err := cmd.Run()
        if err != nil {
                return fmt.Errorf("error enabling timer: %w", err)
        }

        cmd = exec.Command("systemctl", "--user", "start", timerFileName)
        err = cmd.Run()
        if err != nil {
                return fmt.Errorf("error starting timer: %w", err)
        }

        return nil
}

func disableAndStopUserTimer(username string) error {
        timerFileName := fmt.Sprintf(timerFileName, username)

        cmd := exec.Command("systemctl", "--user", "stop", timerFileName)
        err := cmd.Run()
        if err != nil {
                return fmt.Errorf("error stopping timer: %w", err)
        }

        cmd = exec.Command("systemctl", "--user", "disable", timerFileName)
        err = cmd.Run()
        if err != nil {
                return fmt.Errorf("error disabling timer: %w", err)
        }
        return nil
}

func removeUserSystemdFiles(username string) error {
        serviceFileName := fmt.Sprintf(serviceFileName, username)
        timerFileName := fmt.Sprintf(timerFileName, username)

    userConfigDir, err := os.UserConfigDir()
    if err != nil {
        return fmt.Errorf("error getting user config dir: %w", err)
    }

    systemdDir := filepath.Join(userConfigDir, "systemd", "user")


        serviceFilePath := filepath.Join(systemdDir, serviceFileName)
        timerFilePath := filepath.Join(systemdDir, timerFileName)

        err = os.Remove(serviceFilePath)
        if err != nil {
                return fmt.Errorf("error removing service file: %w", err)
        }
        err = os.Remove(timerFilePath)
        if err != nil {
                return fmt.Errorf("error removing timer file: %w", err)
        }
        return nil
}


func isRepoPubliclyAccessible(repoURL string) bool {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        cmd := exec.CommandContext(ctx, "git", "ls-remote", repoURL, "HEAD")
        err := cmd.Run()
        return err == nil
}

func cloneRepo(repoURL, repoPath string) error {
        cmd := exec.Command("git", "clone", repoURL, repoPath)
        err := cmd.Run()
        if err != nil {
                return fmt.Errorf("error during cloning: %w", err)
        }
        return nil
}

func syncRepo(repoPath string) error {
        // Check if it's a git repository
        _, err := os.Stat(filepath.Join(repoPath, ".git"))
        if os.IsNotExist(err) {
                return fmt.Errorf("not a git repository: %s", repoPath)
        }

        cmd := exec.Command("git", "pull", "--ff-only")
        cmd.Dir = repoPath
        output, err := cmd.CombinedOutput()
        if err != nil {
                return fmt.Errorf("error during pull: %w, output: %s", err, string(output))
        }

        if strings.Contains(string(output), "Already up to date") {
                fmt.Println("Already up to date")
                return nil
        }
        return nil
}

func colorize(text, color string) string {
        var colorCode string
        switch color {
        case "green":
                colorCode = "\033[32m"
        case "red":
                colorCode = "\033[31m"
        default:
                return text
        }
        return colorCode + text + "\033[0m"
}