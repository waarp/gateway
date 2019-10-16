Consulter un transfert
======================

.. http:get:: /api/transfers/(int:transfer_id)

   Renvoie les informations du transfert portant l'identifiant ``transfer_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/transfers/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Les informations du transfert ont été renvoyées avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le transfert demandé n'existe pas

   :resjson number id: L'identifiant unique du transfert
   :resjson number ruleID: L'identifiant de la règle de transfert
   :resjson number remoteID: L'identifiant du partenaire de transfert
   :resjson number accountID: L'identifiant du compte de transfert
   :resjson string source: Le chemin d'origine du fichier
   :resjson string destination: Le chemin de destination du fichier
   :resjson date start: La date de début du transfert
   :resjson string status: Le statut actuel du transfert (*PLANNED* ou *TRANSFER*)

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 97

          {
            "id": 1,
            "ruleID": 1,
            "remoteID": 1,
            "accountID": 1,
            "source": "chemin/source/fichier1",
            "destination": "chemin/dest/fichier1",
            "start": "2019-01-01T02:00:00+02:00",
            "status": "TRANSFER"
          }