
selfdir=$(dirname $0)

vcpkg_install_root=$selfdir

set -x
vcpkg --x-install-root=$selfdir --overlay-ports=$selfdir/ports "$@"

