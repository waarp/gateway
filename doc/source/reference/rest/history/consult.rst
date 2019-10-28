Consulter une entrée de l'historique
====================================

.. http:get:: /api/history/(int:history_id)

   Renvoie les informations du transfert portant l'identifiant ``history_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/history/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Les informations du transfert ont été renvoyées avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le transfert demandé n'existe pas

   :resjson number id: L'identifiant unique du transfert
   :resjson string rule: Le nom de la règle de transfert
   :resjson string source: Le nom de l'émetteur du transfert
   :resjson string dest: Le nom du destinataire du transfert
   :resjson string protocol: Le protocole utilisé pour le transfert
   :resjson string filename: Le nom du fichier transféré
   :resjson date start: La date de début du transfert
   :resjson date stop: La date de fin du transfert
   :resjson string status: Le statut final du transfert (*DONE* ou *ERROR*)

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 148

          {
            "id": 1,
            "rule": "regle_sftp",
            "source": "compte_sftp",
            "dest": "serveur_sftp",
            "protocol": "sftp",
            "filename": "nom/de/fichier",
            "start": "2019-01-01T01:00:00+02:00",
            "stop": "2019-01-01T02:00:00+02:00",
            "status": "DONE"
          }