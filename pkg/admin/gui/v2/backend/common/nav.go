package common

import (
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// navEntry is a screen of the sidebar. It is shown to a user who holds the
// right the server requires to read the page, so that the sidebar never
// leads to a refusal; the server checks that right again on the page.
type navEntry struct {
	menu string          // constants.SidebarMenu*
	icon string          // icon of the kit sprite
	key  string          // catalogue key of the label
	path string          // path of the page under the web interface prefix
	perm model.PermsMask // right required to read the page; zero for any authenticated user
}

// navGroup is a collapsible section of the sidebar.
type navGroup struct {
	section string // constants.SidebarSection*
	key     string // catalogue key of the title
	entries []navEntry
}

// navKeyPrefix prefixes the catalogue keys of the screen map; the rest of a
// key names the screen (ScreenEntry.Key), for the pages that say more about
// a screen (the home page).
const navKeyPrefix = "nav."

// navigation is the screen map of the web interface, in the order of the
// glossary. The screens not reworked yet lead to their historic page.
func navigation() []navGroup {
	return []navGroup{
		{section: constants.SidebarSectionConnection, key: "nav.connections", entries: []navEntry{
			{constants.SidebarMenuServers, "server", "nav.servers", "/servers", model.PermServersRead},
			{constants.SidebarMenuClients, "arrows", "nav.clients", "/clients", model.PermServersRead},
			{constants.SidebarMenuPartners, "globe", "nav.partners", "/partners", model.PermPartnersRead},
		}},
		{section: constants.SidebarSectionTreatments, key: "nav.treatments", entries: []navEntry{
			{constants.SidebarMenuRules, "list", "nav.rules", "/transfer_rules_management", model.PermRulesRead},
			{constants.SidebarMenuCryptoKeys, "key", "nav.keys", "/keys", model.PermAdminRead},
			{constants.SidebarMenuSMTPCreds, "envelope", "nav.smtp", "/smtp_credentials_management", model.PermRulesRead},
			{
				constants.SidebarMenuEmailTemplates, "envelope", "nav.email_templates", "/email_templates_management",
				model.PermRulesRead,
			},
		}},
		{section: constants.SidebarSectionMonitoring, key: "nav.supervision", entries: []navEntry{
			{constants.SidebarMenuStatus, "shield", "nav.services", "/services", 0},
			{constants.SidebarMenuTransfers, "clock", "nav.transfers", "/transfer_monitoring", model.PermTransfersRead},
		}},
		{section: constants.SidebarSectionAdministration, key: "nav.administration", entries: []navEntry{
			{constants.SidebarMenuUsers, "users", "nav.users", "/users", model.PermUsersRead},
			{
				constants.SidebarMenuAuthorities, "shield", "nav.authorities", "/authorities",
				model.PermAdminRead,
			},
			{constants.SidebarMenuClouds, "cloud", "nav.clouds", "/clouds", model.PermAdminRead},
			{constants.SidebarMenuSNMP, "sliders", "nav.snmp", "/snmp", model.PermAdminRead},
			{
				constants.SidebarMenuOverrides, "sliders", "nav.overrides", "/overrides",
				model.PermAdminRead,
			},
		}},
	}
}

// ScreenEntry is a screen of the web interface the user may read, as the
// screen map gives it: the sidebar and the home page build on it, so that
// both show the same screens with the same rights.
type ScreenEntry struct {
	Section  string // constants.SidebarSection*
	Menu     string // constants.SidebarMenu*
	Key      string // name of the screen in the catalogue keys ("servers" for "nav.servers")
	Icon     string
	LabelKey string // catalogue key of the label
	URL      string
}

// ScreenSection is a section of the screen map with the screens the user
// may read; a section left without any screen is not listed.
type ScreenSection struct {
	Key      string // constants.SidebarSection*
	TitleKey string // catalogue key of the title
	Screens  []ScreenEntry
}

// Screens lists the screens the user may read, by section, in the order of
// the sidebar.
func Screens(perms model.PermsMask) []ScreenSection {
	groups := navigation()
	sections := make([]ScreenSection, 0, len(groups))

	for _, group := range groups {
		section := ScreenSection{Key: group.section, TitleKey: group.key}

		for _, entry := range group.entries {
			if !perms.HasPermission(entry.perm) {
				continue
			}

			section.Screens = append(section.Screens, ScreenEntry{
				Section:  group.section,
				Menu:     entry.menu,
				Key:      strings.TrimPrefix(entry.key, navKeyPrefix),
				Icon:     entry.icon,
				LabelKey: entry.key,
				URL:      WebuiURL(entry.path),
			})
		}

		if len(section.Screens) > 0 {
			sections = append(sections, section)
		}
	}

	return sections
}

// makeNav builds the sidebar of a user: the entries the user may read, the
// active one marked, without the sections left empty.
func makeNav(translate gui.Translator, perms model.PermsMask, section, menu string) []gui.NavGroup {
	sections := Screens(perms)
	nav := make([]gui.NavGroup, 0, len(sections))

	for _, s := range sections {
		items := make([]gui.NavItem, 0, len(s.Screens))

		for _, screen := range s.Screens {
			items = append(items, gui.NavItem{
				Icon:    screen.Icon,
				Label:   translate(screen.LabelKey),
				URL:     screen.URL,
				Current: screen.Section == section && screen.Menu == menu,
			})
		}

		nav = append(nav, gui.NavGroup{Key: s.Key, Title: translate(s.TitleKey), Items: items})
	}

	return nav
}
