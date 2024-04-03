Modifier une autorité
=====================

.. http:patch:: /api/authorities/(string:authority_name)

   Met à jour l'autorité demandée avec les informations renseignées en JSON.
   Les champs non-spécifiés resteront inchangés.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string name: Le nom de l'autorité
   :reqjson string type: Le type d'autorité (TLS, SSH...)
   :reqjson string publicIdentity: La valeur d'identité publique (certificat,
      clé publique...) de l'autorité
   :reqjson array validHosts: La liste des hôtes que l'autorité est habilitée à
      authentifier. Si vide, l'autorité peut authentifier tous les hôtes.

   :statuscode 201: L'autorité a été remplacée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de l'autorité sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: L'autorité demandée n'existe pas

   :resheader Location: Le chemin d'accès à l'autorité modifiée


   |

   **Exemple de requête**

      .. code-block:: http

         PATCH https://my_waarp_gateway.net/api/authorities/tls_ca HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 123

         {
           "name": "local_tls_ca",
           "validHosts": ["127.0.0.1"]
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/authorities/local_tls_ca