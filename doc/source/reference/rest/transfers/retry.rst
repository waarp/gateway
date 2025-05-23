Reprogrammer un transfert
=========================

.. http:put:: /api/transfer/(int:transfer_id)/retry

   Reprogramme le transfert portant l'identifiant ``transfer_id``. Reprogrammer
   un transfert crée un nouveau transfert identique et l'ajoute à la liste des
   transferts en attente. Seuls les transferts terminés peuvent être reprogrammés.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param date: Fixe la date de démarrage du transfert. La date doit être
      renseignée en format ISO 8601 tel qu'il est spécifié dans la :rfc:`3339`.
      Par défaut, le transfert redémarre immédiatement.
   :type date: date

   :statuscode 201: Le transfert a été reprogrammé avec succès
   :statuscode 400: Le transfert demandé ne peut pas être redémarré
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le transfert demandé n'existe pas

   :resheader Location: Le chemin d'accès au transfert redémarré


   **Exemple de requête**

   .. code-block:: http

      PUT https://my_waarp_gateway.net/api/transfers/1/restart?date=2019-01-01T00:00:00+01:00 HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/transfers/2
