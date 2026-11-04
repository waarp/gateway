Annuler plusieurs transferts
============================

.. http:delete:: /api/transfers

   Annule plusieurs transferts d'un coup, de deux manières.

   Avec le seul paramètre ``target``, annule tous les transferts ayant le
   statut renseigné, comme auparavant : la réponse est un ``202`` sans
   décompte.

   Avec les filtres de la liste (``rule``, ``status``, ``start``, ``followID``,
   ``except`` ou ``ids``), annule les transferts qui les remplissent : ceux que
   :http:get:`/api/transfers` listerait avec les mêmes filtres, moins les
   identifiants exclus par ``except``, ou seulement les identifiants donnés par
   ``ids`` qui remplissent les critères ; ``target``, s'il est renseigné,
   restreint l'ensemble à ses statuts. Les transferts en cours sont arrêtés
   avant d'être annulés ; un transfert terminé, un transfert qui s'exécute sur
   un autre nœud ou qui ne s'arrête pas à temps est compté avec sa raison. La
   réponse est un ``200`` qui compte ce qui a été fait.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param target: Filtre les transferts à annuler suivant leur statut.
      Valeurs possibles : ``planned``, ``running``, ``paused``, ``interrupted``,
      ``error``, et ``all``. Requis en l'absence de tout autre paramètre.
   :type target: string
   :param rule: Restreint l'ensemble aux transferts de la règle renseignée.
      Peut être renseigné plusieurs fois pour plusieurs règles.
   :type rule: string
   :param status: Restreint l'ensemble aux transferts ayant le statut renseigné
      (``PLANNED``, ``RUNNING``, ``PAUSED``, ``INTERRUPTED`` ou ``ERROR``).
      Peut être renseigné plusieurs fois.
   :type status: string
   :param start: Restreint l'ensemble aux transferts dont la date de début est
      postérieure ou égale à celle renseignée (:rfc:`3339`).
   :type start: date
   :param followID: Restreint l'ensemble aux transferts ayant l'ID de flux
      renseigné.
   :type followID: string
   :param except: Exclut de l'ensemble le transfert portant cet identifiant.
      Peut être renseigné plusieurs fois. Incompatible avec ``ids``.
   :type except: int
   :param ids: Restreint l'ensemble aux transferts portant ces identifiants,
      parmi ceux des critères ; un identifiant hors des critères est ignoré.
      Peut être renseigné plusieurs fois. Incompatible avec ``except``.
   :type ids: int

   :statuscode 200: L'ensemble désigné par les filtres a été traité ; la
      réponse compte ce qui a été fait
   :statuscode 202: Les transferts du statut ``target`` ont été annulés
   :statuscode 400: Un ou plusieurs des paramètres de la requête sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 403: L'utilisateur n'a pas le droit d'écriture sur les transferts

   :resjson number done: Le nombre de transferts annulés (réponse ``200``)
   :resjson number refused: Le nombre de transferts laissés de côté
   :resjson array refusals: Les transferts laissés de côté, chacun avec son
      identifiant (``id``) et la raison (``reason``) : ``state`` (un transfert
      terminé ne s'annule plus), ``missing`` (le transfert n'existe plus),
      ``elsewhere`` (le transfert s'exécute sur un autre nœud, seul à pouvoir
      l'annuler) ou ``busy`` (le transfert en cours ne s'est pas arrêté à temps)

   **Exemple de requête (par statut)**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/transfers?target=error HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 202 ACCEPTED

   **Exemple de requête (par filtres)**

   .. code-block:: http

      DELETE https://my_waarp_gateway.net/api/transfers?target=error&rule=commandes&except=12 HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json

      {
        "done": 3,
        "refused": 0
      }
