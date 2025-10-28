import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 10,        // Simulate 10 concurrent virtual users
  duration: '30s', // Run the test for 30 seconds
  thresholds: {
    // Fail the test if 95% of requests take longer than 1 second
    http_req_duration: ['p(95)<1000'], 
  },
};

export default function () {
  // Replace '1' with a valid contest ID from your seeded database
  const res = http.get('http://localhost:8080/api/contests/1/leaderboard');
  
  check(res, { 'status was 200': (r) => r.status == 200 });
  sleep(1); // Each virtual user waits 1 second between requests
}
