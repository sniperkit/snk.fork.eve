from datetime import date
from enum import IntEnum
from os import path
from typing import Optional, NamedTuple

import nltk
import numpy as np
from nltk.stem.snowball import GermanStemmer

from bot.model_definitions import Patterns
from .data import Request, Gender
from .logger import logger
from .static_answers import get_static_answer
from .trainer import load_model

dir = path.dirname(__file__)

# Create German snowball stemmer
stemmer = GermanStemmer()
# Threshold for pattern recognition
ERROR_THRESHOLD = 0.9


class PredictionResult(NamedTuple):
	"""
	Data type for prediction results, used by detect_category
	"""
	mode: IntEnum
	probability: float


def analyze_input(text: str, Mode):
	"""
	Scans the supplied request for pre-defined patterns.

	Args:
		request: The request to scan for patterns.

	Returns:
		The category of the recognized pattern or None if none was found.
		:param Mode:
	"""

	# Load model and data
	model, data = load_model(Mode)
	# Tokenize pattern
	words = nltk.word_tokenize(text)
	stems = [stemmer.stem(word.lower()) for word in words]
	total_stems = data.total_stems
	bag = [0] * len(total_stems)
	for stem in stems:
		for i, s in enumerate(total_stems):
			if s == stem:
				bag[i] = 1

	# Convert to matrix
	input_data = np.asarray([bag])

	# Predict category
	results = model.predict(input_data)[0]
	lower_bound = -1
	if Mode == Patterns:
		lower_bound = 0

	results = [PredictionResult(Mode(i), p) for i, p in enumerate(results)
			   if i > lower_bound]
	results.sort(key=lambda result: result.probability, reverse=True)

	logger.debug('Results: {}'.format(results))

	if len(results) > 0 and results[0].probability > ERROR_THRESHOLD:
		return results[0]

	return None


def answer_for_pattern(request: Request) -> Optional[str]:
	"""
	Scans the supplied request for pre-defined patterns and returns a
	pre-defined answer if possible.

	Args:
		request: The request to scan for patterns.

	Returns:
		A pre-defined answer for the scanned request or None if a pre-defined
		answer isn't possible.
	"""
	category = analyze_input(request.text, Patterns)
	if category is not None:
		# Pattern found, retrieve pre-defined answer
		return get_static_answer(category.mode, request)

	return None


def demo(mode: str):
	"""
	Demo mode for the pattern recognizer
	"""

	request = Request(
		text=input('Please enter a question: '),
		previous_text='Ich bin ein Baum',
		mood=0.0,
		affection=0.0,
		bot_gender=Gender.APACHE,
		bot_name='Lara',
		bot_birthdate=date(1995, 10, 5),
		bot_favorite_color='grün'
		)
	answer = answer_for_pattern(request)
	if answer is None:
		print('No answer found')
	else:
		print('Answer: {}'.format(answer))
