from .logger import logger
from .data import parse_request
from .request_handler import handle_request
import json


def run_loop():
    """
    Starts a request loop that reads lines from stdin.

    Each line represents a new request in JSON format that will be parsed by the
    loop and handled by the request handler. The response returned by the
    request handler will again be formatted as a JSON string and written to
    stdout, including a newline character after every response.
    If an error is raised during parsing of the request data or the request
    handling itself, the current request will be aborted which is signaled by
    the 'error\n' string written to stdout. The loop will then wait for a new
    request.

    The loop can be interrupted by either closing the stdin pipe, resulting in
    an EOFError handled by the loop, or by sending a
    keyboard interrupt (Ctrl + C).
    """

    logger.debug('Starting loop')
    while True:
        try:
            json_data = input()
            request = parse_request(json_data)
            response = handle_request(request)
            print(json.dumps(response))
        except EOFError:
            # Stdin pipe has been closed by Go
            return
        except KeyboardInterrupt:
            # Interrupt requested by developer
            return
        except Exception as ex:
            logger.error('{}: {}'.format(type(ex).__name__, str(ex)))
            # Pass error to Go and await next request
            print('error')
            continue