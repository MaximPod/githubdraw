package main

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"io/ioutil"

	"golang.org/x/image/bmp"
)

const (
	bmpWidth  = 52
	bmpHeight = 7
)

func isWhite(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r>>8 == 255 && g>>8 == 255 && b>>8 == 255
}

func readBMP(path string) ([][]bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, err := bmp.Decode(file)
	if err != nil {
		return nil, err
	}
	pixels := make([][]bool, bmpHeight)
	for y := 0; y < bmpHeight; y++ {
		pixels[y] = make([]bool, bmpWidth)
		for x := 0; x < bmpWidth; x++ {
			c := img.At(x, y)
			pixels[y][x] = !isWhite(c)
		}
	}
	return pixels, nil
}

func getStartSunday() time.Time {
	now := time.Now()
	// Найти ближайшее прошедшее воскресенье
	daysSinceSunday := int(now.Weekday())
	if daysSinceSunday != 0 {
		now = now.AddDate(0, 0, -daysSinceSunday)
	}
	// Минус 52 недели
	start := now.AddDate(0, 0, -7*51) // 52 недели, но первая неделя — текущая
	return start
}

func makeCommit(date time.Time, repoPath string, commitNum int, day string) error {
	graphPath := filepath.Join(repoPath, "graph.md")
	var content string
	if _, err := os.Stat(graphPath); err == nil {
		b, err := ioutil.ReadFile(graphPath)
		if err != nil {
			return fmt.Errorf("failed to read graph.md: %w", err)
		}
		content = string(b)
	}

	// Найти или создать секцию для дня
	lines := strings.Split(content, "\n")
	found := false
	for i, line := range lines {
		if line == day+":" {
			// Добавить новый пункт в список
			lines = append(lines[:i+1], append([]string{fmt.Sprintf("- commit #%d", commitNum)}, lines[i+1:]...)...)
			found = true
			break
		}
	}
	if !found {
		// Добавить новую секцию
		lines = append(lines, day+":")
		lines = append(lines, fmt.Sprintf("- commit #%d", commitNum))
	}
	newContent := strings.Join(lines, "\n")
	if err := ioutil.WriteFile(graphPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write graph.md: %w", err)
	}

	// git add graph.md
	cmdAdd := exec.Command("git", "add", "graph.md")
	cmdAdd.Dir = repoPath
	if err := cmdAdd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	msg := fmt.Sprintf("GitHubDraw: %s commit #%d", date.Format("2006-01-02"), commitNum)
	os.Setenv("GIT_AUTHOR_DATE", date.Format(time.RFC3339))
	os.Setenv("GIT_COMMITTER_DATE", date.Format(time.RFC3339))
	cmd := exec.Command("git", "commit", "--allow-empty", "-m", msg)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func createBranch(repoPath string) (string, error) {
	branchName := "draw-" + time.Now().Format("2006-01-02-15-04-05")
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create branch: %w", err)
	}
	return branchName, nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: GitHubDraw <bmp-file> <repo-path>")
		os.Exit(1)
	}
	bmpPath := os.Args[1]
	repoPath, _ := filepath.Abs(os.Args[2])
	pixels, err := readBMP(bmpPath)
	if err != nil {
		log.Fatalf("Failed to read BMP: %v", err)
	}
	_, err = createBranch(repoPath)
	if err != nil {
		log.Fatalf("Failed to create branch: %v", err)
	}
	start := getStartSunday()
	commitCount := 0
	for x := 0; x < bmpWidth; x++ {
		for y := 0; y < bmpHeight; y++ {
			if pixels[y][x] {
				commitDate := start.AddDate(0, 0, x*7+y)
				commitCount++
				log.Printf("Commit at: %s", commitDate.Format(time.RFC3339))
				day := commitDate.Format("2006-01-02")
				err := makeCommit(commitDate, repoPath, commitCount, day)
				if err != nil {
					log.Printf("Failed to commit at %s: %v", commitDate, err)
				}
			}
		}
	}
}
