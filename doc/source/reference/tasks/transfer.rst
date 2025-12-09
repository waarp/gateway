.. _reference-tasks-transfer:

########
TRANSFER
########

.. note:: À ne pas confondre avec la tâche *PREREGISTER*. Cette dernière permet
   de pré-enregistrer des transferts **serveur** qui s'exécuteront quand un
   partenaire en fera la demande.

Le traitement ``TRANSFER`` programme un nouveau transfert client sur la même
instance de Gateway avec une date de démarrage immédiate. Les arguments sont :

* ``synchronous`` (*boolean*) - Indique si le nouveau transfert doit être exécuté
  de façon synchrone ou asynchrone par rapport au transfert en cours. Par défaut,
  le nouveau transfert sera asynchrone.
* ``file`` (*string*) - Le chemin du fichier à transférer.
* ``output`` (*string*) - Le chemin destination du fichier transféré. Par défaut,
  le nom d'origine du fichier est utilisé.
* ``using`` (*string*) - Le client à utiliser pour le transfert.
* ``to`` (*string*) - Le nom du partenaire auquel se connecter.
* ``as`` (*string*) - Le nom du compte avec lequel s'authentifier auprès du partenaire.
* ``rule`` (*string*) - Le nom de la règle à utiliser pour le transfert.
* ``using`` (*string*) - Le nom du client à utiliser pour faire le transfert.
  Si omit, un client par défaut sera utilisé (si possible).
* ``copyInfo`` (*boolean*) - Indique si les informations du transfert en cours
  doit être copiées sur le nouveau transfert programmé.
* ``info`` (*object*) - Les informations de transfert du nouveau transfert. Si
  les informations du transfert en cours ont été copiées sur le nouveau transfert
  (via le paramètre **copyInfo** décris ci-dessus), les nouvelles informations
  indiquées ici viendront s'additionner à celles-ci (et les écraseront en cas
  de conflit).
* ``nbOfAttempts`` (*number*) - Le nombre de fois que le transfert sera automatiquement
  re-tenté en cas d'échec.
* ``firstRetryDelay`` (*number*) - Le délai entre le transfert original et la première
  reprise automatique. Les unités acceptées sont ``h`` (heures), ``m`` (minutes) et
  ``s`` (secondes), par exemple "1h30m15s". Ne peut être inférieur à 1s.
* ``retryIncrementFactor`` (*number*) - Le facteur par lequel le délai ci-dessus sera
  multiplié à chaque nouvelle tentative. Les nombres décimaux sont acceptés.
* ``timeout`` (*string*) - La durée limite pour le transfert en mode synchrone.
  Passé cette durée, le transfert sera interrompu, et la tâche retournera une erreur.
  N'a pas d'effet pour les transferts asynchrones. Les unités de temps acceptées
  sont : ``s`` (secondes), ``m`` (minutes), et ``h`` (heures).

Mode synchrone et asynchrone
============================

La tâche TRANSFER permet de spécifié si le nouveau transfert doit être exécuté
de façon synchrone ou asynchrone.

En mode synchrone, le transfert est exécuté immédiatement dans le cadre de la tâche
elle-même. Cela signifie que la tâche ne terminera que lorsque le nouveau transfert
aura également terminé. En conséquence, le statut final de la tâche (et du transfert
en cours) dépendra donc du statut final du nouveau transfert.

En mode asynchrone, le transfert est simplement enregistré en base de données pour
être exécuté en différé. Dans ce cas de figure, la tâche ne fait donc que valider
si les informations de transferts sont correctes. La réussite (ou l'échec) du
nouveau transfert n'aura donc pas d'influence sur le statut de la tâche, et donc
du transfert en cours.

Il est à noter que, en mode synchrone, de par le fait que le transfert original
ne terminera pas tant que le rebond n'aura pas également terminé, la connexion
du transfert en cours restera donc ouverte (et inactive) pendant toute la durée
du nouveau transfert. Il est donc déconseillé d'utiliser le mode synchrone dans
le cadre du transfert de gros fichiers.