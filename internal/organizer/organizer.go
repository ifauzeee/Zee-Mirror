package organizer

import (
	"path/filepath"
	"strings"
)

var Categories = map[string][]string{
	"🎥 Videos": {
		".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v",
		".mpg", ".mpeg", ".m2v", ".m2ts", ".mts", ".vob", ".ogv", ".qt",
		".rm", ".rmvb", ".asf", ".amv", ".m2p", ".3gp", ".3g2", ".f4v",
		".mxf", ".r3d", ".braw", ".ari", ".arriraw", ".cineform", ".avchd",
		".divx", ".xvid", ".swf", ".gifv", ".mng", ".roq", ".nsv", ".svi",
		".mpv", ".mp2", ".mpe", ".mpv2", ".ts", ".rec", ".mod", ".tod",
		".vro", ".tp", ".trp", ".m2t", ".mpls", ".bdmv", ".clpi", ".cpi",
		".evo", ".fli", ".flc", ".flic", ".h264", ".h265", ".hevc", ".mjpg",
		".mjpeg", ".nuv", ".ogm", ".pva", ".vp6", ".vp7", ".vp8", ".vp9",
		".wtv", ".xesc", ".y4m", ".yop", ".264", ".265", ".av1",
	},
	"🎵 Music": {
		".mp3", ".flac", ".alac", ".wav", ".ogg", ".m4a", ".aac", ".wma",
		".opus", ".mka", ".dsf", ".dff", ".aiff", ".aif", ".aifc", ".tta",
		".tak", ".wv", ".wvp", ".ofr", ".ofs", ".mpc", ".mp+", ".mpp",
		".ac3", ".dts", ".eac3", ".thd", ".truehd", ".pcm", ".bwf", ".rf64",
		".w64", ".snd", ".au", ".caf", ".sd2", ".sds", ".ircam", ".voc",
		".xi", ".it", ".s3m", ".xm", ".umx", ".mtm", ".669", ".far",
		".med", ".okt", ".stm", ".amr", ".awb", ".3ga", ".m4r", ".m4b",
		".m4p", ".aa", ".aax", ".act", ".dss", ".dvf", ".gsm", ".iklax",
		".ivs", ".mmf", ".msv", ".nmf", ".nsf", ".sln", ".wve", ".kar",
		".sib", ".ly", ".gym", ".vgm", ".psf", ".minipsf", ".gsf",
		".psf2", ".qsf", ".spc", ".nsfe", ".sid", ".nst", ".m15", ".stx",
		".wow", ".uni", ".psm", ".ult", ".okta", ".dmf",
	},
	"📸 Images": {
		".jpg", ".jpeg", ".jpe", ".jif", ".jfif", ".jfi", ".png", ".gif",
		".bmp", ".dib", ".webp", ".tiff", ".tif", ".ico", ".cur", ".heic",
		".heif", ".heics", ".heifs", ".hif", ".avci", ".avcs", ".avif",
		".avifs", ".jxl", ".webp2", ".cr2", ".cr3", ".crw", ".cs1", ".nrw",
		".arw", ".srf", ".sr2", ".rw2", ".pef", ".raf", ".mrw",
		".mdc", ".x3f", ".dcr", ".kdc", ".dcs", ".drf", ".k25", ".rwl",
		".3fr", ".fff", ".mef", ".mos", ".iiq", ".srw", ".erf",
		".bay", ".crwl", ".rwz", ".rw1", ".arq", ".svgz", ".hdr", ".rgbe",
		".xyze", ".pam", ".pcx", ".pct", ".pict", ".pnm", ".pbm",
		".pgm", ".ppm", ".pxr", ".sct", ".sgi", ".rgb", ".bw", ".int",
		".inta", ".picio", ".sun", ".sunras", ".ras", ".im1", ".im8", ".im24",
		".im32", ".rast", ".sixel", ".xbm", ".xpm", ".xwd", ".uyvy",
		".hdp", ".wdp", ".jxr", ".mpo", ".jps", ".pns", ".pat", ".jng",
		".wbmp", ".ilbm", ".lbm", ".iff", ".cel", ".pix", ".spr", ".anb",
		".anm", ".hrp", ".mix", ".neo", ".pac", ".pi1", ".pi2", ".pi3",
		".pi4", ".pi5", ".pi6", ".pi7", ".pi9", ".pntg", ".rle", ".sep",
		".taac", ".tim", ".vicar", ".viff", ".xface", ".xiff", ".ximag",
	},
	"📄 Documents": {
		".pdf", ".doc", ".docx", ".docm", ".dot", ".dotx", ".dotm", ".docb",
		".odt", ".ott", ".odm", ".oth", ".txt", ".rtf", ".md", ".markdown",
		".mdown", ".mkd", ".mkdn", ".mdwn", ".mdtxt", ".mdtext", ".rmd",
		".org", ".adoc", ".asciidoc", ".textile", ".rdoc", ".wiki", ".creole",
		".mediawiki", ".fountain", ".latex", ".ltx", ".ctx", ".sty", ".cls",
		".dtx", ".ins", ".bib", ".bst", ".bbl", ".blg", ".aux", ".toc", ".lof",
		".lot", ".ind", ".ilg", ".glo", ".gls", ".ist", ".hwp", ".hwpx", ".hwt",
		".kwd", ".ksp", ".kpr", ".zabw", ".sxw", ".stw", ".sxc", ".stc", ".sxi",
		".sti", ".sxd", ".std", ".sxg", ".sxm", ".wps", ".wpd", ".wks", ".wdb",
		".abw", ".awt", ".ans", ".ascii", ".utxt",
	},
	"📊 Spreadsheets": {
		".xls", ".xlsx", ".xlsm", ".xlsb", ".xlt", ".xltx", ".xltm", ".xla",
		".xlam", ".xll", ".xlw", ".xlr", ".ods", ".ots", ".fods", ".uos",
		".gsheet", ".numbers", ".numbers-tef", ".gnumeric", ".gnm",
		".wk1", ".wk3", ".wk4", ".123", ".wq1", ".wq2", ".wb1", ".wb2",
		".wb3", ".qpw", ".xlk", ".xls5", ".csv", ".tsv", ".tab", ".dif",
		".slk", ".sylk",
	},
	"🖼️ Presentations": {
		".ppt", ".pptx", ".pptm", ".pot", ".potx", ".potm", ".ppa", ".ppam",
		".pps", ".ppsx", ".ppsm", ".sldx", ".sldm", ".pa", ".odp", ".otp",
		".fodp", ".uop", ".key", ".key-tef", ".gslides", ".pez", ".shw",
		".show", ".shf", ".prz", ".prs", ".dps", ".kpt",
	},
	"📖 Ebooks": {
		".epub", ".mobi", ".azw", ".azw3", ".azw4", ".kf8", ".kfx", ".prc",
		".tpz", ".azw1", ".pobi", ".lit", ".lrf", ".lrx", ".tr2", ".tr3",
		".cbr", ".cbz", ".cb7", ".cbt", ".cba", ".cbg", ".cbn", ".fb2",
		".ibooks", ".iba", ".opf", ".ncx", ".snb", ".tcr", ".pdb", ".pml",
		".pmlz", ".mbp", ".tan", ".imp", ".webz", ".ebm", ".newton", ".aeh",
		".tr", ".etd", ".lrp",
	},
	"📦 Archives": {
		".zip", ".zipx", ".rar", ".r00", ".r01", ".r02", ".rev", ".7z",
		".001", ".002", ".003", ".tar", ".gz", ".gzip", ".tgz", ".bz",
		".bz2", ".bzip", ".bzip2", ".tbz", ".tbz2", ".tb2", ".xz", ".txz",
		".lz", ".lzma", ".tlz", ".lzo", ".z", ".Z", ".tar.Z", ".taz",
		".taZ", ".tz", ".lzh", ".lha", ".arj", ".arc", ".ark", ".zoo",
		".sqz", ".sit", ".sitx", ".sea", ".ice", ".ace", ".alz", ".xxe",
		".b64", ".mim", ".hqx", ".cpt", ".pit", ".pf", ".sar", ".vsix",
		".nupkg", ".snupkg", ".gem", ".whl", ".egg", ".mpkg", ".msm", ".msp",
		".snap", ".flatpak", ".appimage", ".pak", ".pk3", ".pk4", ".vpk",
		".gcf", ".bsp", ".ncf", ".vmt", ".vtf", ".xp3", ".rgssad",
		".rgss2a", ".rgss3a", ".wolf", ".ypf", ".xnb", ".assets", ".resS",
		".tar.gz", ".tar.bz2", ".tar.xz", ".tar.lz", ".tar.lzma",
		".tar.lzo", ".tar.zst", ".tar.sz", ".zst", ".br", ".lz4", ".paq",
		".paq6", ".paq8f", ".paq8jd", ".paq8l", ".paq8o", ".zpaq", ".rz",
		".sfpack", ".uc0", ".uc2", ".ucn", ".uca", ".uha", ".zz", ".pea",
		".pim", ".dar", ".dgc", ".ha", ".kgb", ".lbr", ".lqr", ".rk", ".sda",
		".sen", ".shar", ".shk", ".wim", ".esd", ".swm", ".xar",
	},
	"💿 Disk Images": {
		".iso", ".isz", ".vmdk", ".vdi", ".vhd", ".vhdx", ".hdd", ".hds",
		".qcow", ".qcow2", ".qed", ".vfd", ".avhd", ".avhdx", ".parallels",
		".bin", ".cue", ".mdf", ".mds", ".mdx", ".nrg", ".cdi", ".b5t",
		".b6t", ".b5i", ".b6i", ".bwt", ".bws", ".bwi", ".ccd", ".daa",
		".dao", ".tao", ".uif", ".gbi", ".gi", ".pdi", ".nri", ".ima",
		".imz", ".fcd", ".flp", ".dmg", ".cdr", ".toast", ".sparseimage",
		".sparsebundle", ".sdi", ".pqi", ".vcd", ".ashdisc", ".asd", ".c2d",
		".cfs", ".cso", ".dax", ".dcp", ".disk", ".fdi", ".gdrive", ".ims",
		".imgc", ".lcd", ".mbi", ".pxi", ".rif", ".vaporcd",
	},
	"🛠️ Applications": {
		".exe", ".msi", ".msix", ".appx", ".appxbundle", ".msixbundle",
		".com", ".hta", ".gadget", ".application", ".app", ".bundle",
		".deb", ".rpm", ".run", ".apk", ".apks", ".apkm", ".xapk", ".apkx",
		".aab", ".ipa", ".jad", ".war", ".ear", ".air", ".crx", ".xpi",
		".pet", ".pup", ".sfs", ".slp",
	},
	"📜 Subtitles": {
		".srt", ".sbv", ".ass", ".ssa", ".vtt", ".idx", ".stl", ".ttml",
		".dfxp", ".sami", ".rt", ".usf", ".cap", ".scc", ".mcc", ".itt",
		".sup", ".pgs", ".s2k", ".son", ".ssf", ".pjs", ".jss",
		".dks", ".zeg", ".ovr", ".pan", ".fpc", ".ulp", ".aqt", ".cvd",
		".fab", ".mks", ".phf", ".pin", ".prt", ".sbt", ".spumux",
		".sst", ".sts", ".tds", ".tts", ".vsf",
	},
	"💻 Development": {
		".html", ".htm", ".xhtml", ".mhtml", ".mht", ".maff", ".css", ".scss",
		".sass", ".less", ".styl", ".stylus", ".mjs", ".cjs", ".jsx",
		".tsx", ".vue", ".svelte", ".astro", ".marko", ".riot", ".tag",
		".php3", ".php4", ".php5", ".php7", ".phps", ".phtml", ".asp",
		".aspx", ".ashx", ".asmx", ".axd", ".cshtml", ".vbhtml", ".jsp",
		".jspx", ".jhtml", ".erb", ".rhtml", ".haml", ".slim", ".coffee",
		".ls", ".es", ".es6", ".es7", ".dart", ".elm", ".cc", ".cpp",
		".cxx", ".c++", ".cp", ".hh", ".hpp", ".hxx", ".h++", ".hp",
		".tcc", ".inl", ".ipp", ".ixx", ".csx", ".fs", ".fsx", ".fsi",
		".class", ".kts", ".ktm", ".scala", ".sc", ".groovy", ".gvy",
		".gy", ".gsh", ".gradle", ".clj", ".cljs", ".cljc", ".edn",
		".pyw", ".pyx", ".pyd", ".pyi", ".pyc", ".pyo", ".pyz", ".pyzw",
		".py3", ".ipynb", ".rbw", ".rake", ".gemspec", ".ru", ".builder",
		".thor", ".jbuilder", ".sum", ".work", ".rlib", ".pch",
		".storyboard", ".xib", ".xcworkspace", ".xcodeproj", ".xaml",
		".pl", ".pm", ".pod", ".cgi", ".rdata", ".rds", ".rda", ".rnw",
		".lua", ".rockspec", ".wlua", ".a51", ".nasm", ".yasm", ".pgsql",
		".psql", ".mysql", ".plsql", ".tsql", ".udf", ".ddl", ".dml",
		".json", ".json5", ".jsonl", ".ndjson", ".geojson", ".yaml", ".yml",
		".xsl", ".xslt", ".xsd", ".dtd", ".rdf", ".rss", ".atom", ".opml",
		".mathml", ".toml", ".cfg", ".cmake", ".make", ".makefile", ".ninja",
		".bazel", ".bzl", ".workspace", ".ant", ".maven", ".mill", ".git",
		".gitattributes", ".gitmodules", ".gitkeep", ".hg", ".hgignore",
		".svn", ".glsl", ".hlsl", ".cg", ".shader", ".vert", ".frag", ".geom",
		".tesc", ".tese", ".comp", ".rgen", ".rint", ".rahit", ".rchit",
		".rmiss", ".rcall", ".mesh", ".task", ".wgsl", ".metal", ".ml", ".mli",
		".mll", ".mly", ".hs", ".lhs", ".purescript", ".purs", ".dhall", ".idr",
		".agda", ".lean", ".coq", ".nim", ".nimble", ".nims", ".zig",
		".vlang", ".odin", ".crystal", ".cr", ".d", ".di", ".ada", ".adb",
		".ads", ".f", ".f90", ".f95", ".f03", ".f08", ".for", ".ftn",
		".fpp", ".pas", ".pp", ".lpr", ".dpr", ".dfm", ".lfm", ".ocaml",
		".erl", ".hrl", ".ex", ".exs", ".eex", ".leex", ".heex", ".scilla",
		".clar", ".clarity", ".proto", ".protobuf", ".tf", ".tfvars",
		".hcl", ".nomad", ".vault", ".consul", ".packer", ".vagrant",
		".vagrantfile", ".dockerfile", ".containerfile", ".k8s", ".helm",
		".kustomization", ".graphql", ".gql", ".graphqls", ".prisma",
		".raml", ".wadl", ".wsdl", ".openapi", ".swagger",
	},
	"🔤 Fonts": {
		".ttf", ".tte", ".otf", ".otc", ".woff", ".woff2", ".woff3", ".eot",
		".pfb", ".pfm", ".afm", ".pfr", ".pfa", ".fnt", ".bdf", ".pcf",
		".snf", ".hbf", ".pmf", ".dfont", ".suit", ".pk", ".gf", ".mf",
		".ofm", ".vlw", ".fea", ".ufo", ".glyphs", ".glyphx",
		".fog", ".vfb", ".sfd",
	},
	"🧊 3D Models": {
		".obj", ".fbx", ".dae", ".3ds", ".blend", ".ply", ".gltf", ".glb",
		".usd", ".usda", ".usdc", ".usdz", ".max", ".ma", ".mb", ".c4d",
		".lwo", ".lws", ".lxo", ".modo", ".zpr", ".ztl", ".zbr", ".mud",
		".scn", ".unity", ".unitypackage", ".prefab", ".ue", ".uasset",
		".umap", ".upk", ".t3d", ".pcd", ".e57", ".jt", ".3dxml", ".acis",
		".brep", ".aet", ".wrl", ".wrz", ".x3d", ".x3dv", ".x3db", ".x3dz",
		".vrml", ".3mf", ".amf", ".ac", ".an8", ".aoi", ".b3d", ".bvh",
		".cob", ".csm", ".irrmesh", ".ldr", ".md2", ".md3", ".md5anim",
		".md5camera", ".md5mesh", ".mesh.xml", ".mot", ".mr", ".ndo",
		".nif", ".off", ".ogex", ".pmd", ".pmx", ".q3o", ".q3s",
		".shape", ".skeleton.xml", ".ter", ".uc", ".vox", ".vta", ".vtx",
		".vwx", ".world", ".xsi",
	},
	"🏗️ CAD": {
		".dwf", ".dwfx", ".dws", ".dwt", ".dst", ".slddrt", ".sldprtdot",
		".sldlfp", ".sldstd", ".ide", ".iLogic", ".catpart", ".catproduct",
		".catdrawing", ".catshape", ".cgr", ".session", ".exp", ".dlv",
		".model", ".rvt", ".rfa", ".rte", ".rft", ".skb", ".3dmbak",
		".f3d", ".f3z", ".pln", ".bpn", ".dgn", ".dgnlib", ".db1",
		".lis", ".xas", ".xpr", ".pwd", ".dft", ".ifczip", ".neu",
		".g", ".gcode", ".cnc", ".tap", ".min", ".mpf", ".spl", ".sch",
		".brd", ".pcb", ".kicad_pcb", ".kicad_mod", ".kicad_sch", ".pcbdoc",
		".schdoc", ".cam", ".drl", ".art", ".pho",
	},
	"🗄️ Databases": {
		".db", ".sqlite", ".sqlite3", ".db3", ".sqlitedb", ".ldf", ".ndf",
		".accde", ".mda", ".mdn", ".mdt", ".mdw", ".accdr", ".accdt",
		".accdw", ".laccdb", ".frm", ".myd", ".myi", ".ibd", ".ibdata",
		".ib_logfile", ".db-shm", ".db-wal", ".db-journal", ".realm",
		".realmx", ".rdb", ".fp7", ".fmp12", ".kexi", ".kexis", ".kexic",
		".fdb", ".4dd", ".4dr", ".4dindy", ".ndb", ".ndd",
	},
	"⚙️ Configurations": {
		".ini", ".inf", ".prop", ".prefs", ".plist", ".desktop", ".service",
		".env", ".env.local", ".env.development", ".env.production", ".env.test",
		".bashrc", ".bash_profile", ".bash_logout", ".profile", ".zprofile",
		".zshenv", ".zlogin", ".zlogout", ".vimrc", ".gvimrc", ".emacs",
		".tmux.conf", ".screenrc", ".inputrc", ".xinitrc", ".xsession",
		".Xresources", ".Xdefaults", ".htaccess", ".htpasswd", ".gitconfig",
		".editorconfig", ".eslintrc", ".prettierrc", ".babelrc", ".npmrc",
		".yarnrc", ".nvmrc", ".rvmrc", ".ruby-version", ".python-version",
		".node-version", ".tool-versions", ".travis.yml", ".gitlab-ci.yml",
		".circleci", ".appveyor.yml", ".jenkinsfile", ".drone.yml",
		".npmignore", ".cvsignore", ".bzrignore",
	},
	"🌊 Torrents": {
		".torrent", ".magnet",
	},
	"🖥️ System Files": {
		".sys", ".drv", ".ocx", ".ax", ".acm", ".bpl", ".tlb", ".olb",
		".dpl", ".ime", ".iec", ".ko", ".vxd", ".dxr", ".prx",
	},
	"🎮 Game Files": {
		".sav", ".save", ".gam", ".srm", ".fcs", ".dsv", ".sgm", ".vbm",
		".nes", ".smc", ".sfc", ".swc", ".gd3", ".gd7", ".dx2",
		".usa", ".eur", ".jap", ".st", ".mgd", ".rom", ".a78",
		".gb", ".gbc", ".n64", ".z64", ".v64", ".sms", ".gg", ".smd",
		".gen", ".32x", ".gcm", ".ciso", ".wbfs", ".wad", ".dol", ".elf",
		".nro", ".nso", ".xci", ".nsp", ".nsz", ".unr", ".ut2", ".ut3",
		".udk", ".mcworld", ".mcpack", ".mcaddon", ".mctemplate", ".mcstructure",
		".schematic", ".nbt", ".dat_old", ".replay", ".dem", ".gci",
		".vmu", ".pss", ".psv",
	},
	"📜 Scripts": {
		".sh", ".bash", ".zsh", ".fish", ".ksh", ".csh", ".tcsh", ".command",
		".tool", ".bat", ".cmd", ".btm", ".ps1", ".psm1", ".applescript",
		".scpt", ".scptd", ".awk", ".sed", ".au3", ".ahk", ".nsi", ".nsh",
		".iss",
	},
	"📋 Log Files": {
		".log", ".log.1", ".log.2", ".out", ".err", ".trace", ".debug",
		".warn", ".crash", ".mdmp", ".stackdump", ".etl", ".evtx", ".evt",
	},
	"🗑️ Temporary Files": {
		".tmp", ".temp", ".~", ".cache", ".crdownload", ".download",
		".partial", ".!ut", ".bc!", ".filepart",
	},
	"💾 Backups": {
		".backup", ".bkp", ".bk", ".orig", ".swp", ".swo", ".~undo-tree~",
	},
	"🔒 Encrypted": {
		".aes", ".gpg", ".pgp", ".p7m", ".p7s", ".p7b", ".p7c", ".p7r",
		".cer", ".crt", ".der", ".pfx", ".p12", ".keystore", ".jks",
		".encrypted", ".enc", ".crypt", ".lock", ".kdbx", ".1password",
		".psafe3", ".agilekeychain",
	},
	"✉️ Email": {
		".eml", ".emlx", ".msg", ".oft", ".ost", ".pst", ".mbox", ".mbx",
		".dbx", ".pab", ".wab", ".contact", ".group",
	},
	"📅 Calendar": {
		".ics", ".ical", ".ifb", ".icalendar",
	},
	"👥 Contacts": {
		".vcf", ".vcard", ".ldif", ".abbu",
	},
	"🎨 Vector Graphics": {
		".svg", ".eps", ".ai", ".cdr", ".wmf", ".emf", ".cgm", ".sk", ".sk1",
		".drw", ".cmx", ".pes",
	},
	"🎹 Audio Projects": {
		".aup", ".aup3", ".als", ".ptx", ".logic", ".band", ".rpp",
		".cwp", ".omf", ".aaf", ".sesx", ".stf", ".wfm", ".npr", ".omfi",
		".wrk", ".all", ".drm", ".nwc", ".sf2", ".sf3", ".sfz", ".sfark",
		".dls", ".gig",
	},
	"🎬 Video Projects": {
		".fcpbundle", ".fcpxml", ".fcpxmld", ".drp", ".veg", ".vf", ".kdenlive",
		".mlt", ".wlmp", ".motn", ".imovieproj", ".rcproject", ".vpj",
		".dmsm", ".dmsd",
	},
	"🖌️ Design Projects": {
		".psd", ".psb", ".pdd", ".ait", ".indt", ".indl", ".indb", ".inx",
		".idml", ".fig", ".xd", ".afdesign", ".afphoto", ".afpub", ".cdt",
		".xjt", ".kra", ".krz", ".sai", ".sai2", ".clip", ".pxd", ".pxa", ".pxm",
	},
	"🩺 Medical": {
		".dcm", ".dicom", ".nii", ".nii.gz", ".mnc", ".mgh", ".mgz",
		".nrrd", ".nhdr", ".mha", ".mhd", ".vtk", ".fdf",
	},
	"🔬 Scientific": {
		".mat", ".hdf", ".hdf4", ".hdf5", ".h4", ".h5", ".he4", ".he5",
		".fits", ".fts", ".sdf", ".mol", ".mol2", ".xyz", ".cif", ".spa",
		".spg", ".dta", ".dx", ".jdx", ".opj", ".por",
	},
	"🗺️ GIS": {
		".shp", ".shx", ".prj", ".sbn", ".sbx", ".cpg", ".qix", ".kml",
		".kmz", ".gpx", ".gml", ".gdb", ".ecw", ".jp2", ".mrf", ".vrt",
	},
	"🎧 Playlists": {
		".pls", ".asx", ".wax", ".wvx", ".b4s", ".xspf", ".aimppl", ".pla",
		".smi", ".zpl", ".vlc", ".fpl", ".wpl", ".kpl", ".m4u", ".mpcpl",
	},
	"📋 Metadata": {
		".diz", ".ion", ".sfv", ".md5", ".sha1", ".sha256", ".sha512",
		".par", ".par2",
	},
}

func GetTargetFolder(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return ""
	}
	priority := []string{
		"🎥 Videos", "🎵 Music", "📸 Images", "📄 Documents", "📊 Spreadsheets",
		"🖼️ Presentations", "📖 Ebooks", "🌊 Torrents", "📦 Archives", "💿 Disk Images",
		"🛠️ Applications", "📜 Subtitles", "💻 Development", "🔤 Fonts", "🧊 3D Models",
		"🏗️ CAD", "🗄️ Databases", "⚙️ Configurations", "📜 Scripts", "🎮 Game Files",
		"🖥️ System Files", "📋 Log Files", "🗑️ Temporary Files", "💾 Backups",
		"🔒 Encrypted", "✉️ Email", "📅 Calendar", "👥 Contacts", "🎨 Vector Graphics",
		"🎹 Audio Projects", "🎬 Video Projects", "🖌️ Design Projects", "🩺 Medical",
		"🔬 Scientific", "🗺️ GIS", "🎧 Playlists", "📋 Metadata",
	}
	for _, folder := range priority {
		if extensions, ok := Categories[folder]; ok {
			for _, e := range extensions {
				if ext == e {
					return folder
				}
			}
		}
	}
	return ""
}

func isInCategory(filename, category string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	parts := strings.SplitN(category, " ", 2)
	cleanCategory := parts[len(parts)-1]

	extensions, ok := Categories[category]
	if !ok {
		for k, v := range Categories {
			if strings.HasSuffix(k, cleanCategory) {
				extensions = v
				ok = true
				break
			}
		}
	}
	if !ok {
		return false
	}
	for _, e := range extensions {
		if ext == e {
			return true
		}
	}
	return false
}

func IsVideoFile(filename string) bool { return isInCategory(filename, "🎥 Videos") }
func IsMusicFile(filename string) bool { return isInCategory(filename, "🎵 Music") }
func IsImageFile(filename string) bool { return isInCategory(filename, "📸 Images") }
func IsDocumentFile(filename string) bool {
	return isInCategory(filename, "📄 Documents") ||
		isInCategory(filename, "📊 Spreadsheets") ||
		isInCategory(filename, "🖼️ Presentations")
}
func IsArchiveFile(filename string) bool     { return isInCategory(filename, "📦 Archives") }
func IsApplicationFile(filename string) bool { return isInCategory(filename, "🛠️ Applications") }
func IsSubtitleFile(filename string) bool    { return isInCategory(filename, "📜 Subtitles") }
func IsDevelopmentFile(filename string) bool { return isInCategory(filename, "💻 Development") }
func IsFontFile(filename string) bool        { return isInCategory(filename, "🔤 Fonts") }
func Is3DModelFile(filename string) bool     { return isInCategory(filename, "🧊 3D Models") }
func IsCADFile(filename string) bool         { return isInCategory(filename, "🏗️ CAD") }
func IsDatabaseFile(filename string) bool    { return isInCategory(filename, "🗄️ Databases") }
func IsConfigurationFile(filename string) bool {
	return isInCategory(filename, "⚙️ Configurations")
}
func IsSpreadsheetFile(filename string) bool  { return isInCategory(filename, "📊 Spreadsheets") }
func IsPresentationFile(filename string) bool { return isInCategory(filename, "🖼️ Presentations") }
func IsEbookFile(filename string) bool        { return isInCategory(filename, "📖 Ebooks") }
func IsTorrentFile(filename string) bool {
	return isInCategory(filename, "🌊 Torrents") ||
		strings.HasSuffix(strings.ToLower(filename), ".torrent")
}
func IsDiskImageFile(filename string) bool     { return isInCategory(filename, "💿 Disk Images") }
func IsSystemFile(filename string) bool        { return isInCategory(filename, "🖥️ System Files") }
func IsGameFile(filename string) bool          { return isInCategory(filename, "🎮 Game Files") }
func IsScriptFile(filename string) bool        { return isInCategory(filename, "📜 Scripts") }
func IsLogFile(filename string) bool           { return isInCategory(filename, "📋 Log Files") }
func IsTemporaryFile(filename string) bool     { return isInCategory(filename, "🗑️ Temporary Files") }
func IsBackupFile(filename string) bool        { return isInCategory(filename, "💾 Backups") }
func IsEncryptedFile(filename string) bool     { return isInCategory(filename, "🔒 Encrypted") }
func IsEmailFile(filename string) bool         { return isInCategory(filename, "✉️ Email") }
func IsCalendarFile(filename string) bool      { return isInCategory(filename, "📅 Calendar") }
func IsContactFile(filename string) bool       { return isInCategory(filename, "👥 Contacts") }
func IsVectorGraphicFile(filename string) bool { return isInCategory(filename, "🎨 Vector Graphics") }
func IsAudioProjectFile(filename string) bool  { return isInCategory(filename, "🎹 Audio Projects") }
func IsVideoProjectFile(filename string) bool  { return isInCategory(filename, "🎬 Video Projects") }
func IsDesignProjectFile(filename string) bool {
	return isInCategory(filename, "🖌️ Design Projects")
}
func IsMedicalFile(filename string) bool    { return isInCategory(filename, "🩺 Medical") }
func IsScientificFile(filename string) bool { return isInCategory(filename, "🔬 Scientific") }
func IsGISFile(filename string) bool        { return isInCategory(filename, "🗺️ GIS") }
func IsPlaylistFile(filename string) bool   { return isInCategory(filename, "🎧 Playlists") }
func IsMetadataFile(filename string) bool   { return isInCategory(filename, "📋 Metadata") }
