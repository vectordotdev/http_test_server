set -euo pipefail

if [ $# -ne 1 ]; then
  echo "usage: $0 FILENAME.sh"
  exit 1
fi

for cmd in convert jq gnuplot ruby ; do
  if ! command -v "${cmd}" &> /dev/null
  then
    echo "${cmd} could not be found"
    exit
  fi
done

set -o allexport
source "$1"
set +o allexport

HTTP_TEST_NAME="${1##*/}"
HTTP_TEST_NAME="${HTTP_TEST_NAME%.sh}"

VECTOR="${VECTOR:-vector}"
TEST_CMD="${TEST_CMD:-"${VECTOR} -vv --config ${DIR}/concurrency/vector.toml"}"
HTTP_TEST_SERVER="${HTTP_TEST_SERVER:-${DIR}/../http_test_server}"
OUTPUT_DIR="${OUTPUT_DIR:-$(mktemp -d -t vector-XXXXXXXXXX)}"
TEST_TIME=${TEST_TIME:-60} # how many seconds to run test for
HTTP_TEST_DESCRIPTION=${HTTP_TEST_DESCRIPTION:-${HTTP_TEST_NAME}}

# See ../README.md for additional environment variables that can be set to
# control server behavior

export HTTP_TEST_SUMMARY_PATH="${OUTPUT_DIR}/summary.json"
export HTTP_TEST_PARAMETERS_PATH="${OUTPUT_DIR}/parameters.json"

echo "writing output files to $OUTPUT_DIR"
