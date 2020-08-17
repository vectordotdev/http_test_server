#!/usr/bin/env ruby

require 'date'
require 'json'

####
# This transforms a summary file from http_test_server into a dat file for use
# with gnuplot to graph each request as an interval
#
# It reads the summary from STDIN and writes the DAT to stdout
####

# TODO(jesse) Also plot other HTTP statuses?

summary = JSON.parse(STDIN.read)

requests = summary["requests"].select { |r| r["status"] == 204 }.map do |r|
  r["start"] = DateTime.parse(r["start"])
  r["end"] = DateTime.parse(r["end"])
  r
end.sort_by { |r| r["start"] }

active_requests = []
start_time = requests.map{ |r| r["start"] }.min

puts "# non-overlapping index, start offset (ms), end offset (ms)"
requests.each do |r|
  # find an active request that has ended before this one to reuse the index
  i = active_requests.find_index do |active_request|
    active_request["end"] < r["start"]
  end

  if i != nil then
    active_requests[i] = r
    i = i + 1
  else
    active_requests.push(r)
    i = active_requests.length
  end

  # date subtraction gives fraction of days, multiply to get ms
  rstart = (r["start"] - start_time) * 60 * 60 * 24 * 1000
  rend =  (r["end"] - start_time) * 60 * 60 * 24 * 1000

  puts "%d,%d,%d" % [i, rstart, rend]
end
