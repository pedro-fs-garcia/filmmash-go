package view

import "text/template"

var TemplateCache = make(map[string]*template.Template)

func init() {
	TemplateCache["filmPage"] = template.Must(template.ParseFS(TemplatesFS,
		"template/base.html",
		"template/film.html",
	))
	TemplateCache["filmModal"] = template.Must(template.ParseFS(TemplatesFS,
		"template/film_modal.html",
	))
	TemplateCache["filmVotes"] = template.Must(template.ParseFS(TemplatesFS,
		"template/film_votes.html",
	))
	TemplateCache["duelPage"] = template.Must(template.ParseFS(TemplatesFS,
		"template/base.html",
		"template/index.html",
		"template/duelCard.html",
		"template/filmCard.html",
	))
	TemplateCache["duelCard"] = template.Must(template.ParseFS(TemplatesFS,
		"template/duelCard.html",
		"template/filmCard.html",
	))
	TemplateCache["filmListPage"] = template.Must(template.ParseFS(TemplatesFS,
		"template/base.html",
		"template/filmList.html",
		"template/film_list_items.html",
		"template/film_list_item.html",
	))
	TemplateCache["filmListItems"] = template.Must(template.ParseFS(TemplatesFS,
		"template/film_list_items.html",
		"template/film_list_item.html",
	))
	TemplateCache["filmSearchResults"] = template.Must(template.ParseFS(TemplatesFS,
		"template/film_search_results.html",
		"template/film_list_item.html",
	))
	TemplateCache["myVotesPage"] = template.Must(template.ParseFS(TemplatesFS,
		"template/base.html",
		"template/my_votes.html",
	))
	TemplateCache["adminDashboardPage"] = template.Must(template.ParseFS(TemplatesFS,
		"template/base.html",
		"template/admin_dashboard.html",
		"template/admin_users.html",
	))
	TemplateCache["adminUsers"] = template.Must(template.ParseFS(TemplatesFS,
		"template/admin_users.html",
	))
	TemplateCache["adminTmdbResults"] = template.Must(template.ParseFS(TemplatesFS,
		"template/admin_tmdb_results.html",
	))
	TemplateCache["adminTmdbAddResult"] = template.Must(template.ParseFS(TemplatesFS,
		"template/admin_tmdb_add_result.html",
	))
}
