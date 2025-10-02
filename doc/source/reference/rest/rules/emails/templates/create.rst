Créer un nouveau template d'email
=================================

.. http:post:: /api/email/templates

   Crée un nouveau template d'email.

   :reqheader Authorization: Les identifiants de l'utilisateur.

   :statuscode 201: Le template a été créé avec succès.
   :statuscode 400: Requête invalide.
   :statuscode 401: Authentification d'utilisateur invalide.

   :reqjson string name: Le nom du template.
   :reqjson string subject: Le sujet de l'email.
   :reqjson string mimeType: Le type MIME de l'email. Typiquement soit
     "text/plain" ou "text/html". Par défaut, "text/plain" est utilisé.
   :reqjson string body: Le template du corps de l'email.
   :reqjson array attachments: La liste des fichiers à joindre à l'email.

   :resheader Location: Le chemin d'accès du nouveau template créé

   **Exemple de requête**

   .. code-block:: http

      POST https://my_waarp_gateway.net/api/email/templates HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 238

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

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/email/templates/alerte_erreur
