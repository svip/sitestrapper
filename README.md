Sitestrapper
============

Another static site generator.

Set up
------

Need at least Go 1.11 (using Go modules).

1. Create a directory for your content files, i.e. the 'input directory'.  See further down on how to structure your 
input directory.
2. Create a directory for your webserver's files (where the HTML, CSS, images, etc. will be served from), i.e. the 
'output directory'.

Parameters
----------

* `-i <dir>` specifies the input directory (required).
* `-o <dir>` specifies the output directory (required).
* `--port=<port>` specifies the port for the file server serving the output directory.

If the `port` is not provided, it will not start a server.  This is useful for quickly testing whether your setup works.

Input directory
---------------

The structure of the directory must contain a `site` and `templates` directory.  An optional `media` directory is also 
understood.

Using the `sitemap.yaml` configuration file in the root of the input directory, you can modify the above names of the 
directories, as well as manually describing the site map.  At least a `main` template will be needed as well, see below.

The configuration file has the following settings:

* `pages`: A list of pages on your site.  If not provided, the `site` directory will be assumed to contain these.
* `media`: A list of media, such as stylesheets, etc. for your site.  If not provided, the `media` will be assumed to
contain these.  The type of media will be guessed from the file extension.
* `templates`: A list of templates.  If not provided, the `templates` directory will be assumed to contain these.
* `title`: The title for the site.  If not provided, it will just be empty.
* `categories`: This allows you to order and list the pages by category.  This is useful if you have menus, that need
sorting and categorising.

Generally speaking, it's probably best to provide the least amount for the configuration.  So generally, you should
ignore `pages`, `media` and `templates`, since they can usually be inferred.

Templates
---------

All templates must contain a header, separated by four equal signs on a line (`====`) and then the template itself, for
example:

    name: Main
    ====
    <!doctype html>
    <html>
    ...

The header is written in YAML, and must at least contain the `name` parameter.  A `Main` template is required.  The
names of templates are case insensitive.

Templates use Go's `html/template` package syntax, with the following variables available:

* `.SiteTitle`: The title of the site (see `title` from `sitemap.yaml`).
* `.Title`: The title of the page.
* `.Content`: The actual content of the page.

In addition, the following functions are available:

* `media`: List all media of a certain type (either "stylesheet" or "script).  This is useful for a header, if you wish
to import all your stylesheets, e.g. `{{media "stylesheet"}}`.
* `tmpl`: Import another template.  Like a footer or header, e.g. `{{tmpl "MainHeader"}}`, the name must correspond with
the name of the template (although case insensitive).
* `page`: Gets a single page based on its `id` (see below), so you can link to it with its title, like the below
function.
* `pages`: Returns a list of pages.  If a parameter is provided, it will only list pages from that category, provided in
the configuration's `categories`.  You should range over this list, e.g. 
`{{range pages "SectionA"}}<a href="{{.Link}}">{{.Title}}</a>{{end}}`

Content
-------

In your `site` directory is where you keep your content files.  Like templates, each file must contain a header,
separation of four equal signs (`====`) on its own line and the content, for example:

    title: Index
    id: index
    ====
    Some content...
    ...

The header understands the following parameters:

* `title`: Title of the page.  Can be blank.
* `id`: The identification for the page.  This allows you to refer to the page elsewhere.  If not provided, the path of
  the file will be used to determine the `id`.
* `template`: Which template to use.  If not provided, `main` is assumed.

So far, the content has to be written in [CommonMark Markdown][commonmarkspec].  In the future, more formats will be
supported, but will be inferred by the file extension, so it's wise to name your files `<page>.md` for now.

Markdown links (`[text](uri)`) are modified by the system, if the `uri` starts with one of the following prefixes:

* `id:` modifies the link to refer to the page the ID refers to, e.g. `(text)[id:another page]` -> 
`(text)[/another-page.html]`.
* `image:` modifies the link to add the media image path in front of it.  Right now, it's just `/media/images` (so put 
your images in a directory named `images` inside your media directory), but that will change in the future.  It's useful
when using including pictures, e.g. `![alt](image:logo.png)` -> `![alt](/media/images/logo.png)`.

HTML inside the Markdown is supported.  But must be separated by two line breaks from the regular Markdown, e.g.

    <header>
    
    # Header
    
    </header>

Subdirectories are supported.  Simply create a directory and place pages within them.  The `id` should be unique
throughout the site, and can be referenced from anywhere on the site.

The resulting HTML file path will match that of its original file, e.g. `foo/bar.md` will become `foo/bar.html`.

[commonmarkspec]: https://spec.commonmark.org/