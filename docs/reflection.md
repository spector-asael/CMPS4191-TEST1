# Technical reflection

## Single-image and five-job measurements

The single-image run was acknowledged in approximately 215 ms.
Its queue wait was 198 ms, processing took 7.779 seconds, and
total job duration was 7.977 seconds.

In the five-job burst, all submissions were acknowledged within
approximately 219–228 ms. Queue waits increased from 0.219 seconds
to 31.216 seconds, while processing remained around 7.7 seconds
per image. The last job's total duration was 38.955 seconds.

This demonstrates that 202 Accepted allows the server to acknowledge
accepted work before processing finishes. It does not remove the
processing work or increase the capacity of the single worker.

The worker was configured with an intentional seven-second delay.
These processing measurements include that delay and should not
be interpreted as image-transformation time alone.

Longer-running observations required more polling requests.
Completion detection delays ranged from 68 to 1002 ms in the burst.
The polling interval, request duration, and browser scheduling all
contribute to detection delay.

These results describe one local single-image run and one five-job
burst, rather than a general performance benchmark.

## Limitation of short polling

The browser repeatedly requests status even when the job has not changed.
A longer queue therefore produces more requests. Completion can also
occur between checks, so the browser discovers it later.

A future server-push approach, such as Server-Sent Events, could notify
the browser when state changes and reduce repeated unchanged responses.
It would not make image processing faster or increase worker capacity.
It is outside the scope of this version and has not been implemented.