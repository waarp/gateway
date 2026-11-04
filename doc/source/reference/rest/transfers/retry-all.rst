Reprogrammer plusieurs transferts
=================================

.. http:put:: /api/transfers/retry-all

   Reprogramme les transferts qui remplissent les critères donnés en paramètre
   de requête : ceux que :http:get:`/api/transfers` listerait avec les mêmes
   filtres, moins les identifiants exclus par ``except``, ou seulement les
   identifiants donnés par ``ids`` qui remplissent les critères. Reprogrammer un
   transfert crée un nouveau transfert identique, planifié à la date donnée ou
   immédiatement, comme la reprogrammation d'un seul transfert. Sans aucun
   paramètre, l'ensemble est celui de tous les transferts.

   Un transfert que le modèle refuse (un transfert serveur, un transfert en
   cours ou en attente, un transfert dont la règle, le partenaire, le compte ou
   le client n'existe plus) est compté avec sa raison sans arrêter l'ensemble ;
   une erreur de la base de données ne reprogramme aucun transfert.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param date: Fixe la date de démarrage des nouveaux transferts, au format
      ISO 8601 tel qu'il est spécifié dans la :rfc:`3339`. Par défaut, ils
      démarrent immédiatement.
   :type date: date
   :param rule: Restreint l'ensemble aux transferts de la règle renseignée.
      Peut être renseigné plusieurs fois pour plusieurs règles.
   :type rule: string
   :param status: Restreint l'ensemble aux transferts ayant le statut renseigné
      (``PLANNED``, ``RUNNING``, ``PAUSED``, ``INTERRUPTED``, ``ERROR``,
      ``DONE`` ou ``CANCELLED``). Peut être renseigné plusieurs fois.
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

   :statuscode 200: L'ensemble a été traité ; la réponse compte ce qui a été fait
   :statuscode 400: Un ou plusieurs des paramètres de la requête sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 403: L'utilisateur n'a pas le droit d'écriture sur les transferts

   :resjson number done: Le nombre de transferts reprogrammés
   :resjson number refused: Le nombre de transferts laissés de côté
   :resjson array refusals: Les transferts laissés de côté, chacun avec son
      identifiant (``id``) et la raison (``reason``) : ``server`` (un transfert
      serveur est reprogrammé par le partenaire), ``state`` (un transfert en
      cours ou en attente ne se reprogramme pas), ``missing`` (le transfert
      n'existe plus) ou ``gone`` (sa règle, son partenaire, son compte ou son
      client n'existe plus)
   :resjson array created: Les identifiants des nouveaux transferts, dans
      l'ordre des transferts reprogrammés

   **Exemple de requête**

   .. code-block:: http

      PUT https://my_waarp_gateway.net/api/transfers/retry-all?status=CANCELLED&date=2026-11-05T06:00:00+01:00 HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json

      {
        "done": 2,
        "refused": 0,
        "created": [1041, 1042]
      }
