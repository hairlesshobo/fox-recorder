audio_server:
  auto_start: true
  interface: 
    - coreaudio/ 
  sample_rate: 48000
  frames_per_period: 4096

output:
  directory_template: /Volumes/EOS_DIGITAL/jack/2006-01-02/
  # directory_template: ~/fox_test/2006-01-02/
  # directory_template: /Volumes/JACK/jack/2006-01-02/
  buffer_size_seconds: 20
  minimum_write_size: 0.5
  format: wav
  bit_depth: 16

channels:  
  - ports: 
    - 1
    channel_name: internal_mic
    enabled: true

