audio_server:
  interface: 
    - coreaudio/ 
  sample_rate: 48000
  frames_per_period: 4096
  buffer_size_seconds: 10
  minimum_write_size: 3

output:
  directory: ~/fox_test/2006-01-02/
  format: wav
  bit_depth: 16

channels:  
  - ports: 
    - 1
    channel_name: internal_mic
    enabled: true

