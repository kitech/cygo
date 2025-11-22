# Common Ambient Variables:
#   CURRENT_BUILDTREES_DIR    = ${VCPKG_ROOT_DIR}\buildtrees\${PORT}
#   CURRENT_PACKAGES_DIR      = ${VCPKG_ROOT_DIR}\packages\${PORT}_${TARGET_TRIPLET}
#   CURRENT_PORT_DIR          = ${VCPKG_ROOT_DIR}\ports\${PORT}
#   CURRENT_INSTALLED_DIR     = ${VCPKG_ROOT_DIR}\installed\${TRIPLET}
#   DOWNLOADS                 = ${VCPKG_ROOT_DIR}\downloads
#   PORT                      = current port name (zlib, etc)
#   TARGET_TRIPLET            = current triplet (x86-windows, x64-windows-static, etc)
#   VCPKG_CRT_LINKAGE         = C runtime linkage type (static, dynamic)
#   VCPKG_LIBRARY_LINKAGE     = target library linkage type (static, dynamic)
#   VCPKG_ROOT_DIR            = <C:\path\to\current\vcpkg>
#   VCPKG_TARGET_ARCHITECTURE = target architecture (x64, x86, arm)
#   VCPKG_TOOLCHAIN           = ON OFF
#   TRIPLET_SYSTEM_ARCH       = arm x86 x64
#   BUILD_ARCH                = "Win32" "x64" "ARM"
#   DEBUG_CONFIG              = "Debug Static" "Debug Dll"
#   RELEASE_CONFIG            = "Release Static"" "Release DLL"
#   VCPKG_TARGET_IS_WINDOWS
#   VCPKG_TARGET_IS_UWP
#   VCPKG_TARGET_IS_LINUX
#   VCPKG_TARGET_IS_OSX
#   VCPKG_TARGET_IS_FREEBSD
#   VCPKG_TARGET_IS_ANDROID
#   VCPKG_TARGET_IS_MINGW
#   VCPKG_TARGET_EXECUTABLE_SUFFIX
#   VCPKG_TARGET_STATIC_LIBRARY_SUFFIX
#   VCPKG_TARGET_SHARED_LIBRARY_SUFFIX
#
# 	See additional helpful variables in /docs/maintainers/vcpkg_common_definitions.md

# Also consider vcpkg_from_* functions if you can; the generated code here is for any web accessable
# source archive.
#  vcpkg_from_github
#  vcpkg_from_gitlab
#  vcpkg_from_bitbucket
#  vcpkg_from_sourceforge


# vcpkg_from_github(
#  OUT_SOURCE_PATH SOURCE_PATH
#  REPO antirez/sds
#  HEAD_REF master
#  REF 5347739b1581fcba74fd5cab1fc21d2aef317d71
#  SHA512 cfb0e3d9e953ed18303fb1f07d7723a7c6cfb698ea7ef3bd364130d8374a5298e0b7fcedf513faa61f460c63a903f49445207acd6bfe6a29e8a22352078f63b4
# )

###
set(SUBPKGS "srdja/Collections-C/archive/3920f28431ecf82c9e7e78bbcb60fe473d87edf9.tar.gz"
    "attractivechaos/klib/archive/1979581d3021534c547b46d025851da5cd7c344d.tar.gz"
    "antirez/sds/archive/5347739b1581fcba74fd5cab1fc21d2aef317d71.tar.gz")
foreach(subpkg ${SUBPKGS})
    get_filename_component(arname ${subpkg} NAME)
    vcpkg_download_distfile(ARCHIVE
        URLS https://github.com/${subpkg}
        FILENAME "${arname}"
        SKIP_SHA512
    )
    vcpkg_extract_source_archive(SOURCE_PATH
    ARCHIVE "${ARCHIVE}"
    PATCHES
        # unglue.patch
        # 0100-add-host-tools-dir.diff
    )
    file(GLOB INST_FILES "${SOURCE_PATH}/*.h" "${SOURCE_PATH}/*.c")
    foreach(file ${INST_FILES})
        file(INSTALL ${file} DESTINATION "${CURRENT_PACKAGES_DIR}/include")
    endforeach()
ENDFOREACH()

# file(INSTALL "${SOURCE_PATH}/LICENSE" DESTINATION "${CURRENT_PACKAGES_DIR}/share/${PORT}" RENAME copyright)

# # Check if one or more features are a part of a package installation.
# # See /docs/maintainers/vcpkg_check_features.md for more details
# vcpkg_check_features(OUT_FEATURE_OPTIONS FEATURE_OPTIONS
#   FEATURES
#     tbb   WITH_TBB
#   INVERTED_FEATURES
#     tbb   ROCKSDB_IGNORE_PACKAGE_TBB
# )

# vcpkg_cmake_configure(
#     SOURCE_PATH "${SOURCE_PATH}"
#     # OPTIONS -DUSE_THIS_IN_ALL_BUILDS=1 -DUSE_THIS_TOO=2
#     # OPTIONS_RELEASE -DOPTIMIZE=1
#     # OPTIONS_DEBUG -DDEBUGGABLE=1
# )

# vcpkg_cmake_install()

# # Moves all .cmake files from /debug/share/sds/ to /share/sds/
# # See /docs/maintainers/ports/vcpkg-cmake-config/vcpkg_cmake_config_fixup.md for more details
# When you uncomment "vcpkg_cmake_config_fixup()", you need to add the following to "dependencies" vcpkg.json:
#{
#    "name": "vcpkg-cmake-config",
#    "host": true
#}
# vcpkg_cmake_config_fixup()

# Uncomment the line below if necessary to install the license file for the port
# as a file named `copyright` to the directory `${CURRENT_PACKAGES_DIR}/share/${PORT}`
# vcpkg_install_copyright(FILE_LIST "${SOURCE_PATH}/LICENSE")
