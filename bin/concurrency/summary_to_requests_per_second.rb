#!/usr/bin/env ruby

require 'date'
require 'json'

####
# This transforms a summary file from http_test_server into a dat file of
# requests / s for use with gnuplot
#
# It reads the summary from STDIN and writes the DAT to stdout
####

# TODO(jesse) Also plot other HTTP statuses?

SECONDS = 24 * 60 * 60 # Scale up the parsed date to seconds

summary = JSON.parse(STDIN.read)

requests = summary["requests"].map do |r|
  r["start"] = DateTime.parse(r["start"])
  r["end"] = DateTime.parse(r["end"])

  r
end

start_time = requests.map{ |r| r["start"] }.min
end_time = requests.map{ |r| r["end"] }.max
duration = Integer((end_time - start_time) * SECONDS)

issued_requests = requests.map { |r| Integer((r["start"] - start_time) * SECONDS) }
success_requests = requests.select { |r| r["status"] == 204 }.map { |r| Integer((r["end"] - start_time) * SECONDS) }

puts "# offset (seconds), issued req/sec, success req/sec"
duration.times do |ts|
  num_issued = issued_requests.count { |r| r == ts }
  num_success = success_requests.count { |r| r == ts }

  puts "%d,%d,%d" % [ts * 1000, num_issued, num_success]
end
