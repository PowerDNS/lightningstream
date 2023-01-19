function rsync_prepare() {
  set -x
  if [ -d ~/.ssh ]; then
    return 0
  fi
  mkdir ~/.ssh || return 1
  ssh-keyscan "${RSYNC_HOST}" >> ~/.ssh/known_hosts || return 1
  cp "${RSYNC_KEY}" ~/.ssh/id_rsa || return 1
  ssh-keygen -l -f ~/.ssh/id_rsa
  chmod -R go-rwx ~/.ssh || return 1
  cd "${CI_PROJECT_DIR}" || return 1
  VERSION="$(./builder/gen-version)" || return 1
  export VERSION
  return 0
}

function rsync_rpm_package() {
  if [ -z "${1}" ]; then
    exit 1
  fi
  rsync_prepare || exit 1
  set -x
  cd "${CI_PROJECT_DIR}" || exit 1
  mkdir -p "${VERSION}/${1}" || exit 1
  cp ./builder/tmp/latest/"${1}"/dist/*/*.rpm "${VERSION}/${1}/"
  rm -f latest
  ln -sf "${VERSION}" latest
  rsync --archive --progress "${VERSION}" "${RSYNC_USER}@${RSYNC_HOST}:" || exit 1
  rsync --archive --progress latest "${RSYNC_USER}@${RSYNC_HOST}:" || exit 1
}

function rsync_deb_package() {
  if [ -z "${1}" ]; then
    exit 1
  fi
  rsync_prepare || exit 1
  set -x
  cd "${CI_PROJECT_DIR}" || exit 1
  mkdir -p "${VERSION}/${1}" || exit 1
  cp ./builder/tmp/latest/"${1}"/dist/*.deb "${VERSION}/${1}/"
  rm -f latest
  ln -sf "${VERSION}" latest
  rsync --archive --progress "${VERSION}" "${RSYNC_USER}@${RSYNC_HOST}:" || exit 1
  rsync --archive --progress latest "${RSYNC_USER}@${RSYNC_HOST}:" || exit 1
}

function rsync_sdist_package() {
  rsync_prepare || exit 1
  set -x
  cd "${CI_PROJECT_DIR}" || exit 1
  mkdir -p "${VERSION}/sdist" || exit 1
  cp ./builder/tmp/latest/sdist/* "${VERSION}/sdist/"
  rm -f latest
  ln -sf "${VERSION}" latest
  rsync --relative --archive --progress "${VERSION}/sdist" "${RSYNC_USER}@${RSYNC_HOST}:" || exit 1
  rsync --archive --progress latest "${RSYNC_USER}@${RSYNC_HOST}:" || exit 1
}
