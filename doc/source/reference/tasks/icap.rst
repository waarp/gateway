.. _ref-tasks-icap:

ICAP (BETA)
===========

Le traitement ``ICAP`` upload le fichier de transfert vers un serveur ICAP pour
être traité. Dans le cas où le transfert est un envoi, la méthode Icap ``REQMOD``
sera utilisée. Dans le cas d'une réception, la methode ``RESPMOD`` sera utilisée.

Les arguments de ce traitement sont:

* ``uploadURL`` (*string*) - L'URL complète de la requête ICAP. Cela inclue
  l'hôte, le port, ainsi que le chemin.
* ``useTLS`` (*bool*) - Indique si la connexion au serveur doit se faire en TLS.
  Par défaut, la connexion sera en clair.
* ``timeout`` (*string*) - Le temps de timeout de la requête. Les unités de temps
  acceptées sont: ``ms`` (millisecondes), ``s`` (secondes), ``m`` (minutes),
  et ``h`` (heures).
* ``allowFileModifications`` (*bool*) - Indique si le serveur ICAP est autorisé
  à modifier le fichier de transfert, auquel cas le fichier de transfert sera
  écrasé par le contenu de la réponse du serveur. Par défaut les modifications
  sont interdites.
* ``onError`` (*string*) - L'action à effectuer en cas d'erreur. Les valeurs
  acceptées sont : (vide) pour ne rien faire, ``delete`` pour supprimer
  le fichier de transfert ou ``move`` pour déplacer le fichier de transfert dans
  un autre dossier. Par défaut, aucune action n'est prise sur le fichier en cas
  d'erreur (le transfert tombera tout de même en erreur).
* ``onErrorMovePath`` (*string*) - Le chemin vers lequel le fichier de transfert
  sera déplacé en cas d'erreur. Cette option est obligatoire si ``onError`` est
  ``move``. Dans le cas contraire, cette option n'a aucun effet.