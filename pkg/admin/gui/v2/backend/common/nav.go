package common

import (
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

// navigation is the screen map of the web interface, in the order of the
// glossary. The screens not reworked yet lead to their historic page.
func navigation() []navGroup {
	return []navGroup{
		{section: constants.SidebarSectionConnection, key: "nav.connections", entries: []navEntry{
			{constants.SidebarMenuServers, "server", "nav.servers", "/server_management", model.PermServersRead},
			{constants.SidebarMenuClients, "arrows", "nav.clients", "/local_client_management", model.PermServersRead},
			{constants.SidebarMenuPartners, "globe", "nav.partners", "/partner_management", model.PermPartnersRead},
		}},
		{section: constants.SidebarSectionTreatments, key: "nav.treatments", entries: []navEntry{
			{constants.SidebarMenuRules, "list", "nav.rules", "/transfer_rules_management", model.PermRulesRead},
			{constants.SidebarMenuCryptoKeys, "key", "nav.keys", "/cryptographic_key_management", model.PermAdminRead},
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
			{constants.SidebarMenuUsers, "users", "nav.users", "/user_management", model.PermUsersRead},
			{
				constants.SidebarMenuAuthorities, "shield", "nav.authorities", "/managing_authentication_authorities",
				model.PermAdminRead,
			},
			{constants.SidebarMenuClouds, "cloud", "nav.clouds", "/cloud_instance_management", model.PermAdminRead},
			{constants.SidebarMenuSNMP, "sliders", "nav.snmp", "/snmp_management", model.PermAdminRead},
			{
				constants.SidebarMenuOverrides, "sliders", "nav.overrides", "/managing_configuration_overrides",
				model.PermAdminRead,
			},
		}},
	}
}

// makeNav builds the sidebar of a user: the entries the user may read, the
// active one marked, without the sections left empty.
func makeNav(translate gui.Translator, perms model.PermsMask, section, menu string) []gui.NavGroup {
	groups := navigation()
	nav := make([]gui.NavGroup, 0, len(groups))

	for _, group := range groups {
		items := make([]gui.NavItem, 0, len(group.entries))

		for _, entry := range group.entries {
			if !perms.HasPermission(entry.perm) {
				continue
			}

			items = append(items, gui.NavItem{
				Icon:    entry.icon,
				Label:   translate(entry.key),
				URL:     WebuiURL(entry.path),
				Current: group.section == section && entry.menu == menu,
			})
		}

		if len(items) > 0 {
			nav = append(nav, gui.NavGroup{Key: group.section, Title: translate(group.key), Items: items})
		}
	}

	return nav
}
