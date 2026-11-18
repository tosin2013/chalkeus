package sample;

import static org.junit.jupiter.api.Assertions.assertEquals;

import org.junit.jupiter.api.Test;

// A hidden test for code the item has not written yet: it doesn't compile on the unchanged sample,
// so the smoke install exercises pilot-maven-prefetch's fallback (the probe test).
class FarewellTest {
    @Test
    void farewell() {
        assertEquals("bye", Farewell.say());
    }
}
