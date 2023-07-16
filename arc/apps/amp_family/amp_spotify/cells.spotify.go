package amp_spotify

import (
	"fmt"
	"strings"

	"github.com/arcspace/go-arc-sdk/apis/arc"
	"github.com/arcspace/go-archost/arc/apps/amp_family/amp"
	respot "github.com/arcspace/go-librespot/librespot/api-respot"
	"github.com/zmb3/spotify/v2"
)

type spotifyCell struct {
	amp.CellBase[*appCtx]
	spotifyID spotify.ID
	pinner    func(dst *amp.PinnedCell[*appCtx], cell *spotifyCell) error
	info      arc.CellInfo
}

type playlistCell struct {
	spotifyCell
	amp.MediaPlaylist
}

type trackCell struct {
	spotifyCell
	amp.MediaInfo
	playable *arc.AssetRef // non-nil when pinned
}

func (cell *spotifyCell) ExportAttrs(app *appCtx, dst *arc.AttrBatch) error {
	dst.Add(app.CellInfoAttr, &cell.info)
	return nil
}

func (cell *spotifyCell) WillPinCell(app *appCtx, parent amp.Cell[*appCtx], req arc.CellReq) (string, error) {
	if err := app.waitForSession(); err != nil {
		return "", err
	}
	return cell.CellBase.WillPinCell(app, parent, req)
}

func (cell *spotifyCell) PinInto(dst *amp.PinnedCell[*appCtx]) error {
	return cell.pinner(dst, cell)
}

func (cell *playlistCell) ExportAttrs(app *appCtx, dst *arc.AttrBatch) error {
	cell.spotifyCell.ExportAttrs(app, dst)
	dst.Add(app.MediaPlaylistAttr, &cell.MediaPlaylist)
	return nil
}

func (cell *trackCell) ExportAttrs(app *appCtx, dst *arc.AttrBatch) error {
	cell.spotifyCell.ExportAttrs(app, dst)
	dst.Add(app.MediaInfoAttr, &cell.MediaInfo)
	if cell.playable != nil {
		dst.Add(app.PlayableAssetAttr, cell.playable)
	}
	return nil
}

func (cell *trackCell) PinInto(dst *amp.PinnedCell[*appCtx]) error {
	app := dst.App
	asset, err := app.respot.PinTrack(string(cell.spotifyID), respot.PinOpts{})
	if err != nil {
		return err
	}
	url, err := app.PublishAsset(asset, arc.PublishOpts{
		HostAddr: app.Session().LoginInfo().HostAddr,
	})
	if err != nil {
		return err
	}

	cell.playable = &arc.AssetRef{
		MediaType: asset.MediaType(),
		URI:       url,
	}
	return nil
}

func pin_appHome(dst *amp.PinnedCell[*appCtx], cell *spotifyCell) error {

	// app := cell.app
	// if err := app.waitForSession(); err != nil {
	// 	return err
	// }

	{
		child := addChild_dir(dst, "Followed Playlists")
		child.info.Glyph = &arc.AssetRef{
			URI:    "/icons/ui/providers/playlists.png",
			Scheme: arc.URIScheme_File,
		}
		child.pinner = func(dst *amp.PinnedCell[*appCtx], cell *spotifyCell) error {
			resp, err := dst.App.client.CurrentUsersPlaylists(dst.App)
			if err != nil {
				return err
			}
			for i := range resp.Playlists {
				addChild_Playlist(dst, resp.Playlists[i])
			}
			return nil
		}
	}

	{
		child := addChild_dir(dst, "Followed Artists")
		child.info.Glyph = &arc.AssetRef{
			URI:    "/icons/ui/providers/artists.png",
			Scheme: arc.URIScheme_File,
		}
		child.pinner = func(dst *amp.PinnedCell[*appCtx], cell *spotifyCell) error {
			resp, err := dst.App.client.CurrentUsersFollowedArtists(dst.App)
			if err != nil {
				return err
			}
			for i := range resp.Artists {
				addChild_Artist(dst, resp.Artists[i])
			}
			return nil
		}
	}

	{
		child := addChild_dir(dst, "Recently Played")
		child.info.Glyph = &arc.AssetRef{
			URI:    "/icons/ui/providers/tracks.png",
			Scheme: arc.URIScheme_File,
		}
		child.pinner = func(dst *amp.PinnedCell[*appCtx], cell *spotifyCell) error {
			resp, err := dst.App.client.CurrentUsersTopTracks(dst.App)
			if err != nil {
				return err
			}
			for i := range resp.Tracks {
				addChild_Track(dst, resp.Tracks[i])
			}
			return nil
		}
	}

	{
		child := addChild_dir(dst, "Recently Played Artists")
		child.info.Glyph = &arc.AssetRef{
			URI:    "/icons/ui/providers/artists.png",
			Scheme: arc.URIScheme_File,
		}
		child.pinner = func(dst *amp.PinnedCell[*appCtx], cell *spotifyCell) error {
			resp, err := dst.App.client.CurrentUsersTopArtists(dst.App)
			if err != nil {
				return err
			}
			for i := range resp.Artists {
				addChild_Artist(dst, resp.Artists[i])
			}
			return nil
		}
	}

	{
		child := addChild_dir(dst, "Saved Albums")
		child.info.Glyph = &arc.AssetRef{
			URI:    "/icons/ui/providers/albums.png",
			Scheme: arc.URIScheme_File,
		}
		child.pinner = func(dst *amp.PinnedCell[*appCtx], cell *spotifyCell) error {
			resp, err := dst.App.client.CurrentUsersAlbums(dst.App)
			if err != nil {
				return err
			}
			for i := range resp.Albums {
				addChild_Album(dst, resp.Albums[i].SimpleAlbum)
			}
			return nil
		}

	}

	// CurrentUsersShows

	return nil
}

func addChild_dir(dst *amp.PinnedCell[*appCtx], title string) *spotifyCell {
	cell := &spotifyCell{}
	dst.NewChild(&cell.CellBase, cell)

	cell.info = arc.CellInfo{
		CellDefID: dst.App.LinkCellSpec,
		Title:     title,
	}

	return cell
}

func pin_Playlist(dst *amp.PinnedCell[*appCtx], cell *spotifyCell) error {
	app := dst.App
	resp, err := app.client.GetPlaylistItems(app, cell.spotifyID)
	if err != nil {
		return err
	}
	for _, item := range resp.Items {
		if item.Track.Track != nil {
			addChild_Track(dst, *item.Track.Track)
		} else if item.Track.Episode != nil {
			// TODO: handle episodes
		}
	}
	return nil
}

// func pin_Track(dst *amp.PinnedCell[*appCtx], cell *spotifyCell) error {
// 	app := dst.App
// 	asset, err := app.respot.PinTrack(string(cell.spotifyID), respot.PinOpts{})
// 	if err != nil {
// 		return err
// 	}
// 	assetRef := &arc.AssetRef{
// 		MediaType: asset.MediaType(),
// 	}
// 	assetRef.URI, err = app.PublishAsset(asset, arc.PublishOpts{
// 		HostAddr: app.Session().LoginInfo().HostAddr,
// 	})
// 	if err != nil {
// 		return err
// 	}
// 	cell.(*trackCell).MediaInfo.URI = assetRef.URI
// 	cell.SetAttr(dst.App, amp.Attr_Playable, assetRef)
// 	return nil
// }

func pin_Album(dst *amp.PinnedCell[*appCtx], cell *spotifyCell) error {
	resp, err := dst.App.client.GetAlbum(dst.App, cell.spotifyID)
	if err != nil {
		return err
	}
	for _, track := range resp.Tracks.Tracks {
		addChild_Track(dst, spotify.FullTrack{
			SimpleTrack: track,
			Album:       resp.SimpleAlbum,
		})
	}
	return nil
}

func pin_ArtistAlbums(dst *amp.PinnedCell[*appCtx], cell *spotifyCell) error {
	resp, err := dst.App.client.GetArtistAlbums(dst.App, cell.spotifyID, allAlbumTypes)
	if err != nil {
		return err
	}
	for i := range resp.Albums {
		addChild_Album(dst, resp.Albums[i])
	}
	return nil
}

var allAlbumTypes = []spotify.AlbumType{
	spotify.AlbumTypeAlbum,
	spotify.AlbumTypeSingle,
	spotify.AlbumTypeAppearsOn,
	spotify.AlbumTypeCompilation,
}

func addChild_Playlist(dst *amp.PinnedCell[*appCtx], playlist spotify.SimplePlaylist) {
	cell := &playlistCell{}
	cell.spotifyID = playlist.ID
	cell.pinner = pin_Playlist
	dst.NewChild(&cell.CellBase, cell)

	cell.info = arc.CellInfo{
		CellDefID: dst.App.PlaylistCellSpec,
		Title:     playlist.Name,
		Subtitle:  playlist.Description,
		Link:      chooseBestLink(playlist.ExternalURLs),
	}
	setGlyphs(playlist.Images, &cell.info)

	cell.MediaPlaylist = amp.MediaPlaylist{
		TotalItems: int32(playlist.Tracks.Total),
	}
}

func addChild_Artist(dst *amp.PinnedCell[*appCtx], artist spotify.FullArtist) {
	cell := &spotifyCell{}
	cell.spotifyID = artist.ID
	cell.pinner = pin_ArtistAlbums
	dst.NewChild(&cell.CellBase, cell)

	cell.info = arc.CellInfo{
		CellDefID: dst.App.LinkCellSpec,
		Title:     artist.Name,
		Subtitle:  fmt.Sprintf("%d followers", artist.Followers.Count),
		Link:      chooseBestLink(artist.ExternalURLs),
	}
	setGlyphs(artist.Images, &cell.info)
}

func addChild_Album(dst *amp.PinnedCell[*appCtx], album spotify.SimpleAlbum) {
	cell := &spotifyCell{}
	cell.spotifyID = album.ID
	cell.pinner = pin_Album

	cell.info = arc.CellInfo{
		CellDefID: dst.App.LinkCellSpec,
		Title:     album.Name,
		Subtitle:  formArtistDesc(album.Artists),
		Link:      chooseBestLink(album.ExternalURLs),
	}
	setGlyphs(album.Images, &cell.info)
}

func addChild_Track(dst *amp.PinnedCell[*appCtx], track spotify.FullTrack) {
	if track.IsPlayable != nil && !*track.IsPlayable {
		return
	}
	cell := &trackCell{}
	cell.spotifyID = track.ID
	dst.NewChild(&cell.CellBase, cell)

	artistDesc := formArtistDesc(track.Artists)

	cell.info = arc.CellInfo{
		CellDefID: dst.App.PlayableCellSpec,
		Title:     track.Name,
		Subtitle:  artistDesc,
		About:     track.Album.Name,
		Link:      chooseBestLink(track.ExternalURLs),
	}
	setGlyphs(track.Album.Images, &cell.info)

	cell.MediaInfo = amp.MediaInfo{
		Flags:       amp.HasAudio | amp.IsSeekable | amp.NeedsNetwork,
		Title:       track.Name,
		AuthorDesc:  artistDesc,
		Collection:  track.Album.Name,
		ItemNumber:  int32(track.TrackNumber),
		Duration16:  int64(arc.ConvertMsToTimeFS(int64(track.Duration))),
		CoverArt:    cell.info.Glyph.URI,
		Popularity:  .01 * float32(track.Popularity), // 0..100 => 0..1
		ReleaseTime: track.Album.ReleaseDateTime().Unix(),
	}
}

/**********************************************************
 *  Helpers
 */

func setGlyphs(images []spotify.Image, info *arc.CellInfo) (glyph *arc.AssetRef, cover *arc.AssetRef) {
	info.Glyph = chooseBestImage(images, 200)
	if info.Glyph != nil {
		if len(images) > 1 {
			info.GlyphLarge = chooseBestImage(images, 800)
		}
	}
	return
}

func chooseBestImage(images []spotify.Image, closestSize int) *arc.AssetRef {
	bestImg := -1
	bestDiff := 0x7fffffff

	for i, img := range images {
		diff := img.Width - closestSize

		// If the image is smaller than what we're looking for, make differences matter more
		if diff < 0 {
			diff *= -2
		}

		if diff < bestDiff {
			bestImg = i
			bestDiff = diff
		}
	}
	if bestImg < 0 {
		return nil
	}
	return &arc.AssetRef{
		MediaType: "image/x-spotify",
		URI:       images[bestImg].URL,
		PixWidth:  int32(images[bestImg].Width),
		PixHeight: int32(images[bestImg].Height),
	}
}

func chooseBestLink(links map[string]string) *arc.AssetRef {
	if url, ok := links["spotify"]; ok {
		return &arc.AssetRef{
			URI: url,
		}
	}
	return nil
}

func formArtistDesc(artists []spotify.SimpleArtist) string {
	switch len(artists) {
	case 0:
		return ""
	case 1:
		return artists[0].Name
	default:
		str := strings.Builder{}
		for i, artist := range artists {
			if i > 0 {
				str.WriteString(amp.ListItemSeparator)
			}
			str.WriteString(artist.Name)
		}
		return str.String()
	}
}