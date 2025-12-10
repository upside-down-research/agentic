load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_dependencies():
    go_repository(
        name = "com_github_alecthomas_kong",
        importpath = "github.com/alecthomas/kong",
        sum = "h1:G5diXxc85KvoV2f0ZRVuMsi45IrBgx9zDNGNj165aPA=",
        version = "v0.9.0",
    )
    go_repository(
        name = "com_github_apapsch_go_jsonmerge_v2",
        importpath = "github.com/apapsch/go-jsonmerge/v2",
        sum = "h1:axGnT1gRIfimI7gJifB699GoE/oq+F2MU7Dml6nw9rQ=",
        version = "v2.0.0",
    )
    go_repository(
        name = "com_github_aymanbagabas_go_osc52_v2",
        importpath = "github.com/aymanbagabas/go-osc52/v2",
        sum = "h1:X0S2KqKVsq2jmSlY+wPOmlkuMiGfNL6I2uDP0gPChb8=",
        version = "v2.0.1",
    )
    go_repository(
        name = "com_github_beorn7_perks",
        importpath = "github.com/beorn7/perks",
        sum = "h1:VlbKKnNfV8bJzeqoa4cOKqO6bYr3WgKZxO8Z16+hsOM=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_cespare_xxhash_v2",
        importpath = "github.com/cespare/xxhash/v2",
        sum = "h1:DC2CZ1Ep5Y4k3ZQ899DldepgrayRUGE6BBZ/cd9Cj44=",
        version = "v2.2.0",
    )
    go_repository(
        name = "com_github_charmbracelet_lipgloss",
        importpath = "github.com/charmbracelet/lipgloss",
        sum = "h1:KiIEVLWWC+c6J36YRjmDQieTCBtXwJf5qZ/ovQ2oP9k=",
        version = "v0.10.0",
    )
    go_repository(
        name = "com_github_charmbracelet_log",
        importpath = "github.com/charmbracelet/log",
        sum = "h1:G9bQAcx8rWA2T3pWvx7YtPTPwgqpk7D68BX21IRW8ZM=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_go_logfmt_logfmt",
        importpath = "github.com/go-logfmt/logfmt",
        sum = "h1:wGYYu3uicYdqXVgoYbvnkrPVXkuLM1p1ifugDMEdRi4=",
        version = "v0.6.0",
    )
    go_repository(
        name = "com_github_google_uuid",
        importpath = "github.com/google/uuid",
        sum = "h1:t6JiXgmwXMjEs8VusXIJk2BXHsn+wx8BZdTaoZ5fu7I=",
        version = "v1.3.1",
    )
    go_repository(
        name = "com_github_gorilla_mux",
        importpath = "github.com/gorilla/mux",
        sum = "h1:TuBL49tXwgrFYWhqrNgrUNEY92u81SPhu7sTdzQEiWY=",
        version = "v1.8.1",
    )
    go_repository(
        name = "com_github_influxdata_influxdb_client_go_v2",
        importpath = "github.com/influxdata/influxdb-client-go/v2",
        sum = "h1:+/OzCES099GBDSpwUNjZY5f8khNhKYJb7zvOJKsLSfc=",
        version = "v2.13.0",
    )
    go_repository(
        name = "com_github_influxdata_line_protocol",
        importpath = "github.com/influxdata/line-protocol",
        sum = "h1:0etQ0A02YWdtRRCY0cMN8WXaGcn0KRHuKh+HWi9/zHs=",
        version = "v0.0.0-20200327222509-2487e7298839",
    )
    go_repository(
        name = "com_github_lucasb_eyer_go_colorful",
        importpath = "github.com/lucasb-eyer/go-colorful",
        sum = "h1:1nnpGOrhyZZuNyfu1QjKiUICQ74+3FNCN69Aj6K7nkY=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_mattn_go_isatty",
        importpath = "github.com/mattn/go-isatty",
        sum = "h1:LUXCnvUvSM6FXAsj6nnfc8Q2tp1dIgUfY9Kc8GsSOiQ=",
        version = "v0.0.19",
    )
    go_repository(
        name = "com_github_mattn_go_runewidth",
        importpath = "github.com/mattn/go-runewidth",
        sum = "h1:UNAjwbU9l54TA3KzvqLGxwWjHmMgBUVhBiTjelZgg3U=",
        version = "v0.0.15",
    )
    go_repository(
        name = "com_github_muesli_reflow",
        importpath = "github.com/muesli/reflow",
        sum = "h1:IFsN6K9NfGtjeggFP+68I4chLZV2yIKsXJFNZ+eWh6s=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_muesli_termenv",
        importpath = "github.com/muesli/termenv",
        sum = "h1:twN+ijkXDYJEt+4YMOJoaWBbVtEiMqdl4dAMlXaZQRc=",
        version = "v0.15.2",
    )
    go_repository(
        name = "com_github_oapi_codegen_runtime",
        importpath = "github.com/oapi-codegen/runtime",
        sum = "h1:P4rqFX5fMFWqRzY9M/3YF9+aPSPPB06IzP2P7oOxrWo=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_prometheus_client_golang",
        importpath = "github.com/prometheus/client_golang",
        sum = "h1:ygXvpU1AoN1MhdzckN+PyD9QJOSD4x7kmXYlnfbA6JU=",
        version = "v1.19.0",
    )
    go_repository(
        name = "com_github_prometheus_client_model",
        importpath = "github.com/prometheus/client_model",
        sum = "h1:gQz4mCbXsO+nc9n1hCxHcGA3Zx3Eo+UHZoInFGUIXNM=",
        version = "v0.5.0",
    )
    go_repository(
        name = "com_github_prometheus_common",
        importpath = "github.com/prometheus/common",
        sum = "h1:0VPGssJe6ZUgKCOougn2fZLx+6vyFSg5QY5CbAgAZBU=",
        version = "v0.48.0",
    )
    go_repository(
        name = "com_github_prometheus_procfs",
        importpath = "github.com/prometheus/procfs",
        sum = "h1:jluTpSng7V9hY0O2R9DzzJHYb2xULk9VTR1V1R/k6Bo=",
        version = "v0.12.0",
    )
    go_repository(
        name = "com_github_rivo_uniseg",
        importpath = "github.com/rivo/uniseg",
        sum = "h1:WUdvkW8uEhrYfLC4ZzdpI2ztxP1I582+49Oc5Mq64VQ=",
        version = "v0.4.7",
    )
    go_repository(
        name = "org_golang_x_exp",
        importpath = "golang.org/x/exp",
        sum = "h1:gVJMid8atcGlvLG9UD0oRfb0FfMGB+rtM7FsVhsYaWk=",
        version = "v0.0.0-20231006140011-7918f672742d",
    )
    go_repository(
        name = "org_golang_x_net",
        importpath = "golang.org/x/net",
        sum = "h1:NIXaBOtQ/SIQcQmBpWLM67vbQpCX8X/9xrVjrcfe3fo=",
        version = "v0.20.0",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:xWw16ngr6ZMtmxDyKyIgsE93KNKz5HKmMa3b8ALHidU=",
        version = "v0.16.0",
    )
    go_repository(
        name = "org_golang_google_protobuf",
        importpath = "google.golang.org/protobuf",
        sum = "h1:g0LDEJHgrBl9N9r17Ru3sqWhkIx2NB67okBHPwC7hs8=",
        version = "v1.32.0",
    )
