package feed

type Address struct {
	s string
}

type LoaderName struct {
	s string
}

var (
	RSSLoaderName    = LoaderName{"rss"}
	CausesLoaderName = LoaderName{"causes"}
)

type SavedFeed struct {
	address    Address
	loaderName LoaderName
}

type Storage interface {
	GetFeeds() ([]SavedFeed, error)
}

type LoaderFactory interface {
	GetLoader(loaderName LoaderName) ([]Loader, error)
}

type Loader struct {
}

type Updater struct {
}

type LoadedFeedStorage interface {
}
