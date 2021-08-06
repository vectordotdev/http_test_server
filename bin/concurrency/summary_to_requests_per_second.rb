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

step = (1.to_f/24/60/60) # 1s step (in fraction of days)

summary = JSON.parse(STDIN.read)

requests = summary["requests"].map do |r|
  r["start"] = DateTime.parse(r["start"])
  r["end"] = DateTime.parse(r["end"])

  r
end

start_time = requests.map{ |r| r["start"] }.min
end_time = requests.map{ |r| r["end"] }.max

puts "# offset (ms), req/s"
start_time.step(end_time, step).each do |d|
  num_active = requests.select do |r|
    r["start"] > d and r["start"] < (d + step)
  end.length

  # date subtraction gives fraction of days, multiply to get ms
  puts "%d,%d" % [(d - start_time) * 60 * 60 * 24 * 1000, num_active]
end
