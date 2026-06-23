package playwright

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

// Driver wraps the Playwright interactions
type Driver struct {
	pw      *playwright.Playwright
	browser playwright.BrowserContext
	Page    playwright.Page
}

// NewDriver initializes a Playwright persistent context with the given user data directory
func NewDriver(userDataDir string) (*Driver, error) {
	err := playwright.Install(&playwright.RunOptions{
		SkipInstallBrowsers: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to install playwright dependencies: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}

	options := playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(false),
		Channel:  playwright.String("chrome"),
	}

	browser, err := pw.Chromium.LaunchPersistentContext(userDataDir, options)
	if err != nil {
		if stopErr := pw.Stop(); stopErr != nil {
			return nil, fmt.Errorf("failed to launch persistent context: %v, and failed to stop playwright: %v", err, stopErr)
		}
		return nil, fmt.Errorf("failed to launch persistent context: %w", err)
	}

	pages := browser.Pages()
	var page playwright.Page
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page, err = browser.NewPage()
		if err != nil {
			return nil, fmt.Errorf("could not create new page: %w", err)
		}
	}

	return &Driver{
		pw:      pw,
		browser: browser,
		Page:    page,
	}, nil
}

func (d *Driver) Goto(url string) error {
	_, err := d.Page.Goto(url)
	return err
}

func (d *Driver) Click(selector string) error {
	return d.Page.Locator(selector).Click()
}

func (d *Driver) Type(selector string, text string) error {
	return d.Page.Locator(selector).Fill(text)
}

func (d *Driver) Eval(js string) (interface{}, error) {
	return d.Page.Evaluate(js)
}

func (d *Driver) Close() error {
	if err := d.browser.Close(); err != nil {
		return err
	}
	if err := d.pw.Stop(); err != nil {
		return err
	}
	return nil
}
