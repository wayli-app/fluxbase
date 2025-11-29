/**
 * Test job to verify execution logs are working
 * @fluxbase:timeout 30
 * @fluxbase:memory 128
 */

export default async function handler(request: any) {
  console.log('Starting test job...');
  console.log('This is a test log message 1');
  
  await new Promise(resolve => setTimeout(resolve, 1000));
  
  console.log('This is a test log message 2');
  console.log('Processing data...');
  
  await new Promise(resolve => setTimeout(resolve, 1000));
  
  console.log('Almost done...');
  console.log('Finishing up!');
  
  return {
    success: true,
    message: 'Test completed with logs'
  };
}
