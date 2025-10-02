Consulter un template d'email
=============================

.. http:get:: /api/email/templates/(string:name)

   Renvoie le template d'email demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur.

   :statuscode 200: Le template a été renvoyé avec succès.
   :statuscode 401: Authentification d'utilisateur invalide.
   :statuscode 404: Le template demandé n'existe pas.

   :resjson string name: Le nom du template.
   :resjson string subject: Le sujet de l'email.
   :resjson string mimeType: Le type MIME de l'email. Typiquement soit
     "text/plain" ou "text/html".
   :resjson string body: Le template du corps de l'email.
   :resjson array attachments: La liste des fichiers à joindre à l'email.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/email/templates/alerte_erreur HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 436

      {
        "name": "alerte_erreur",
        "subject": "Alerte erreur de transfert",
        "mimeType": "text/plain",
        "body": "!! ALERTE !!

          Le transfert n°#TRANSFERID# du fichier #TRUEFILENAME#
          a échoué le #DATE# à #HOUR#
          avec le code #ERRORCODE# et le message \"#ERRORMSG#\".

          Waarp",
        "attachments": ["gateway.log"]
      }
