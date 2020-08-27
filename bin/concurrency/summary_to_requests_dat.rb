#!/usr/bin/env ruby

require 'date'
require 'json'

####
# This transforms a summary file from http_test_server into a dat file for use
# with gnuplot to graph each request as an interval
#
# It reads the summary from STDIN and writes the DAT to stdout
####

summary = JSON.parse(STDIN.read)

requests = summary["requests"].map do |r|
  r["start"] = DateTime.parse(r["start"])
  r["end"] = DateTime.parse(r["end"])

  # date subtraction gives fraction of days, multiply to get ms

  # make sure we plot at least 1ms for the request
  if  ((r["end"] - r["start"]) * 60 * 60 * 24 * 1000) < 1 then
    r["end"] = r["end"] + (1.0 / (24 * 60 * 60 * 1000)) * 1
  end
  r
end.sort_by { |r| r["start"] }

active_requests = []
start_time = requests.map{ |r| r["start"] }.min

puts "# non-overlapping index, start offset (ms), end offset (ms), status code"
requests.each do |r|
  i = case r["status"]
      when 200..299
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
      else # put on 0 axis to make them easier to see
        0
      end

  # date subtraction gives fraction of days, multiply to get ms
  rstart = (r["start"] - start_time) * 60 * 60 * 24 * 1000
  rend =  (r["end"] - start_time) * 60 * 60 * 24 * 1000

  puts "%d,%d,%d,%d" % [i, rstart, rend, r["status"]]
end
