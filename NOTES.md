## PART 1 NOTES

## 1.My Interpretation of the problem 
Relay receives event notifications from third-party delivery providers for marketing messages.

Here `Relay means Backend Service

The services needs:
- Accept provider events through `POST/events`.
- Track the four event types:
    - `sent`
    - `delivered`
    - `opened`
    - `clicked`

- Provide campagin-level statistics through:
    -`GET/campaign/{campaign_id}/stats`
- Avoid double-counting events when providers retry the same event.
- Handle events that arrive late or out of order
- keep campaign statistics correct when multiple events are processed cocurrently.
- Provide a simple API response that is useful for a dashboard.

My prmiary goal is to build a small,reliable service with clear behaviour rather than trying to implement every possible feature.


## 2.Assumptions I am making 

- `event_id` is the identifier for one event ,so i will use for duplicate detection.

- If the same id received more than once,i will count it only once

- Duplicate detection should apply across the service,not just within a single batch.

- `timestamp` represents when the event happened at the provider, not when relay received it.

- I will use the timestamp when calculating the time-based statistics

- When statistics are grouped by day, I will use the event's UTC date.

- The event type must be one of the four types defined.

- `metadata` is optional that does not affect the basic campaign statistics.

- I will assume an event with missing or invalid required feilds should not be processed at a valid event.

- I will use the in-memory persistence for the intial implementation because the current traffic is relatively small and it explicity allows either SQLite or in-memoery persistence.
- I will keep the API behaviour simple and predictable.

## 3.Ambiguities I noticed 

- Duplicate events should not be counted twice,but the expected API response when a duplicate is received is not defined.

- The behaviour for a request containing both valid and invalid events 

- It is unclear whether the malformed event should reject the entire request or the remaining events should still be processed.

- The meaning of the `opened` count needs clarification.It means the total number of `opened` events or the number of unique contacts who opened.

- The counting rule for the repeated `clicked` events from same contact is undefined.

- The handling of the same `event_id` arriving with different event information is  not defined.

- The scope of duplicate detection is not stated clearly.

- The choice between the sqlite and in-memory storage is given but the expected behaviour after a service restart is not specified.

- The maximum size of a single `POST/events` request is not mentioned.

- If single event is malformed or invalid,should we reject the entire batch or only mark that particular event as valid?

## 4.Questions I would ask the PM if i could 

- When a duplicate event is received ,should i treat it as a succesfull no-op ,or should the API return that indicates it was already processed.

- If a request contains mostly valid events but one or two are malformed ,should the valid events still be processed?

- Is `event_id` guaranteed to uniquely identify an event across all providers?

- If the same `event_id` arrives again with different information, which event should be considered valid?

- Should repeated clicks from the same contact be counted separately?

- For the `opened` count ,should multiple opens frmo the same contact count multiple items,or should a contact be counted separately?

- When an event arrives late,should it be included based on the date when it actually happened rather than when Relay received it?

- What is the expected maximum number of events that counld be sent in a single request ?

## 5. What I will prioritize in the time available ,and in what order 

I have a limited amount of time for the assignment, so i will focus on getting the required behaviour correct before working on optional improvements.

My order of priority will be:

1. **Get the API WORKING**
    - Define the event structure.
    - Implement `POST /events`.
    - Implement `GET /campaigns/{campaign_id}/stats`.

2. **Handle the event cases correctly**
    - Validate the incoming events.
    - Prevent duplicate events fmo being counted more than once.
    - Make sure ;ate and out-of-order events are handles correctly.

3. **Make concurrent processing reliable**
    - Process events concurrently.
    - Protect shared data from race conditions.
    - Make sure the final statistics remain correct.

4. **Add focused tests**
    - Test duplicate events.
    - Test duplicates across batches.
    - Test invalid input.
    - Test concurrent processing.

5. **Review the implementation**
    - Check error handling.
    - Check memory usage.
    - Make sure the API behaviour is clear and consistent.

6. **Use remaining time for optional work**
    - Consider the optional campaign events endpoint.
    - Make small improvements only if they provide a clear benefit.

I will avpid spending time on frnotend,deployment,authentication, or other work outside the main requirements.





### PART-4 : Scale Memo 

The current implementation uses in-memory storage,which is simple and enough for this assignment. If the event volume becomes very large ,this approach will have some limitations.

## What breaks first?
The main problem is the in-memory data.If the server instances are running ,each instance will  have its own data and duplicate checking will its own data and duplicate checking will not be shared.

## What would I change?
I would move the data to a database such as PostgreSQL and store the events permanently.

I would use a unique constraint on `event_id` so that provider retires do not count the same event twice.

For handling a large number of events, I would put a Queue between the API and the workers.

`API ->Queue ->Workers->Database`

This allows multiple workers to process evets at the same time.

## Campaign statistics 

Instead of calculating statistics from all events whenver the stats API is called,I would maintain campaign counters such as sent,delivered,opened and clicked while processing events.

I would also maintain dailly delivered counts using the event timestamp in UTC.

## Late events

Since events can arrive late or out of order ,I would use the event's timestamp rather than the time when the server received it.A late event should update correct day's statistics.

## Overall 

For this assignment ,the current in0-memory solution is sufficient.Atlarger scale,I would mianly replace it with a database ,add a queue and workers ,and maintain pre-calculated statistics.







## Part5: The Angry Marketer

At 10:00 ,the dashboard shows:

- Sent:1,000,000
- Delivered:975,000
- Opened: 248,000
- Clicked :20,000


Before assuming that the dashboard is broken,I would consider:
- More delivery events may have arrived during these 30 minutes, so delivered increased.
- Some events may have arrived late or out of order.
- Open events may have been deduolicated or corrected.
- The delivered and opened events may be processed at different times.
- There could be a difference between the event timestamp and the time the event was received.
- There could be an issue with the dashboard query ,aggregation ,or cached data.
- There could also be an actual bug in the code.

## What i would check first 

I wpuld first check the raw events and processing logs between 10:00 and 10:30.

I would checl whether the 5,000 additional delivered events were actually received and whether any open events were removed, deduplicated,rejecyed, or processed late.

Then I would compar the dashboard numbers with the actual store/ processed data.

## How I would decide if it is a bug 

If the underlying events and aggregation produce the same numbers as the dashboard ,then the dashboard is probably behaving as expected.

If the underlying data says that opened should still be 250,000 or higher but the dashboard shows 248,000 then new delivery events arriving will naturally increase the number.

The more suspisious change is opened going down fmo 250,000 to 248,000.I check the underlying data and how the metric is calculated.






### What I completed /what I intentionally skipped 

## completed ones

- Compelted the required `POST /events ` API.
- Compelted the `GET /campaign/{campaign_id}/events` API because it was not required.
- Added validation and duplicate event handling 
- Added unique open and daily delivered statistics.
- Added tests for the important cases.
- Fixed all four bugs in the debugging task.
- Checked that the debugging output matches `expected_output.txt`.
- Ran the race detector and verified that there is no data race.

## Intentionally skipped

- I did not implement the optional `GET/campaigns/{campaign_id}/events` API because it was not required.

- I did not work on deployment ,Kubernetes ,cloud setuo ,or authentication because they were not required for this assignment.

## If i had another day 

- I would move the current in-memory storage to a database.

- I would implement the optional events listing API with pagination.
- I would add more tests for large batches, late events, and failure cases.
