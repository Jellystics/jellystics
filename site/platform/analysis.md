# Insights, discover and playback

## Insights

![Insights](/screenshots/en/insights.jpg){.shot}

How people watch:

- **Completion distribution**: views by 10 % buckets; the 100 % peak is finished views. **Finished views by type**.
- **Binges** per series, **pauses** (paused time ÷ watched time per title), **weekly rhythm** heatmap.
- **Session length** (median, false starts excluded), **resumed vs from the start**, **rewatches**.
- **Most skipped intros** (plugin), **genres over time**, **time of day by type**.

## Discover

![Discover](/screenshots/en/discover.jpg){.shot}

What sleeps in the libraries:

- **Never watched**: count, share of the catalog and disk size, oldest additions first.
- **Abandoned**: titles stopped between 10 and 80 % and not played for more than 60 days.
- **Added in the period** and how many were already watched.
- **Storage vs viewing**: file size against views; large files never watched stand out in red.

## Playback

![Playback and transcoding](/screenshots/en/playback.jpg){.shot}

Playback health:

- **Direct play** and **transcode** shares, **top transcode cause**, **most transcoded client**.
- **Play method per day**, **clients** (% transcoded), **false starts by client**.
- **Resolution actually served** (4K, 1080p, 720p, SD) and 4K files served below 4K; **HDR** (native, tone-mapped, SDR).
- **Why it transcodes**: the reasons reported by Jellyfin (container, video/audio/subtitle codec, bitrate, resolution, bit depth, HDR range…).
- **Video or audio only**, hardware acceleration, **codec conversions**.
- **Local or remote** with places (IP geolocation), **bitrate** distribution (average and 95th percentile).

Cards that depend on detailed capture (reasons, network, bitrate) say “Data collected since…” when the data starts inside the period.
