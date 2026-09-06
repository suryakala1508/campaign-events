## BUGS.md

## Bug1 -Duplicate event_id counted across batches

**What:** The same `event_id` could be counted if it appeared in another batch.This caused the counts to become higher than expected.

**Why:** `processBatch()` was creating a new `seen` map every time it was called.Since the events were processed in batches of 200 ,the map was cleared after each batch.Becuase of this,an event that was already processed in an eariler batch could be counted again in a later batch.

**Fix:** I moved the `seenEventID's` map outside `processBatch()` so it is shared across all batches .

**Verified:** Ran `go run .events.jsonl` and the output matched `expected_output.txt`.

---

## Bug2- unique _opens was counted globally 

**What:** `unqique_opens` was lower than expected because the same contact was only counted once across all campaigns

**Why:** The `openedBy` map was using only `ContactID ` as the key.

For example,if the same contact opened two different campaigns,the second campaign was treated as if the cintact had  already opened something.
But `unique_opens` should be unique for each campaign

**Fix:** I changed the key include both the campaign and contact 
where the same contact can be counted once in each different campaign

**Verified:** ```go ev.CampaignID +"\x00" +ev.ContactID

## Bug3:Daily Delivered coutn used local time

**What:** Some delivered events were counted under the wrong date .An extra 2026-08-08 date was appearing

**Why:** The code was using ev.Timestamp.Local() instead of UTC.Becuase of the timezone conversion,some events were moved to the next day.

**Fix:** Changes it to:
        ev.Timestamp.UTC().Format("2006-01-02")

Now the dially coutn always uses the UTC date from the event timestamp.

**Verified:** The dially delivered values now match expected_output.txt and the extra 2026-08-08  date is gone.

## Bug4: Concurrent counter updates

**What:**
 The sent,delivered,opened, and clicked counts could change between runs.

**Why:** There are 8 workers procesing events at the same time.They were updating the sae campaign counters without any synchronization. Because of this,some updates could be lost.

**Fix:** I added a sync.utex to `CampaignStats` and used it while updating the counters in `apply()`
The existing worker pool,channel, and WaitGroup were kept as tehy were.

**Verified:** Ran `go run .events.jsonl` multiple times and got the same output every time.The output also matched `expected_output.txt`.

I also ran:

go run -race.events.jsonl 

It completed completed successfully without any `WARNING :DATA RACING`, and the output matched the expected output 